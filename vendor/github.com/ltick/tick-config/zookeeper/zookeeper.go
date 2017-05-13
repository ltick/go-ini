package zookeeper

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"errors"

	"github.com/samuel/go-zookeeper/zk"
	"github.com/go-ozzo/ozzo-config"
)

var (
	errConnect            = errors.New("zookeeper: connect failed")
	errConfigMissUser     = errors.New("zookeeper: config miss user")
	errConfigMissPassword = errors.New("zookeeper: config miss password")
	errConfigMissHost     = errors.New("zookeeper: config miss host")
	errConfigMissRootPath = errors.New("zookeeper: config empty root_path")
)

// ServiceConfig is a yaml config parser and implements Config interface.
type ZookeeperServiceConfig struct {
	configer *ZookeeperServiceConfiger
}

// Parse returns a ZookeeperServiceConfiger with parsed zookeeper config map.
func (c *ZookeeperServiceConfig) Init(config map[string]string) (config.ServiceConfiger, error) {
	c.configer = &ZookeeperServiceConfiger{
		config: config,
		Mutex:  sync.RWMutex{},
	}
	err := c.configer.Init()
	if err != nil {
		return nil, err
	}
	return c.configer, nil
}

// ZookeeperServiceConfiger A Config represents the yaml configuration.
type ZookeeperServiceConfiger struct {
	conn    *zk.Conn
	config  map[string]string
	errors  chan error
	data    map[string]interface{} // key=>val
	watcher *ZookeeperServiceConfigerWatcher
	cache   *ZookeeperServiceConfigerCache
	Mutex   sync.RWMutex
}

type ZookeeperServiceConfigerWatcher struct {
	keyWatcher map[string]bool
	Mutex      sync.RWMutex
}

type ZookeeperServiceConfigerCache struct {
	cacheTimeout map[string]time.Time
	cacheTime    time.Duration
	Mutex        sync.RWMutex
}

func (c *ZookeeperServiceConfiger) Init() error {
	if c.config["root_path"] == "" {
		return errConfigMissRootPath
	}
	err := c.connect(c.config)
	if err != nil {
		return errConnect
	}
	c.cache = &ZookeeperServiceConfigerCache{
		Mutex: sync.RWMutex{},
	}
	c.cache.cacheTimeout = make(map[string]time.Time)
	if c.config["cache_time"] != "" {
		c.cache.cacheTime, err = time.ParseDuration(c.config["cache_time"])
		if err != nil {
			c.cache.cacheTime = 300 * time.Second // 5min
		}
	} else {
		c.cache.cacheTime = 300 * time.Second // 5min
	}
	c.errors = make(chan error, 1)
	c.watcher = &ZookeeperServiceConfigerWatcher{
		Mutex: sync.RWMutex{},
	}
	c.watcher.keyWatcher = make(map[string]bool)
	data := c.loadData(c.config["root_path"])
	if data != nil {
		switch data.(type) {
		case string:
			c.data = make(map[string]interface{})
		default:
			c.data = data.(map[string]interface{})
		}
		go func() {
			for {
				select {
				case err := <-c.errors:
					if err == zk.ErrSessionExpired {
						c.addAuth(c.config)
					} else {
						//log
					}
				}
			}
		}()
		return nil
	} else {
		err := <-c.errors
		return errors.New("zookeeper: init error!\n" + err.Error())
	}
}

// Bool returns the boolean value for a given key.
func (c *ZookeeperServiceConfiger) Bool(key string) (bool, error) {
	val := c.getData(key)
	if val != nil {
		return config.ParseBool(val)
	}
	return false, errors.New("zookeeper: key '" + key + "' not exist")
}

// DefaultBool return the bool value if has no error
// otherwise return the defaultval
func (c *ZookeeperServiceConfiger) DefaultBool(key string, defaultval bool) bool {
	if v, err := c.Bool(key); err == nil {
		return v
	}
	return defaultval
}

// Int returns the integer value for a given key.
func (c *ZookeeperServiceConfiger) Int(key string) (int, error) {
	val := c.getData(key)
	if val != nil {
		return strconv.Atoi(val.(string))
	}
	return 0, errors.New("not exist key:" + key)
}

// DefaultInt returns the integer value for a given key.
// if err != nil return defaltval
func (c *ZookeeperServiceConfiger) DefaultInt(key string, defaultval int) int {
	if v, err := c.Int(key); err == nil {
		return v
	}
	return defaultval
}

// Int64 returns the int64 value for a given key.
func (c *ZookeeperServiceConfiger) Int64(key string) (int64, error) {
	val := c.getData(key)
	if val != nil {
		return strconv.ParseInt(val.(string), 10, 64)
	}
	return 0, errors.New("not exist key:" + key)
}

// DefaultInt64 returns the int64 value for a given key.
// if err != nil return defaltval
func (c *ZookeeperServiceConfiger) DefaultInt64(key string, defaultval int64) int64 {
	if v, err := c.Int64(key); err == nil {
		return v
	}
	return defaultval
}

// Float returns the float value for a given key.
func (c *ZookeeperServiceConfiger) Float(key string) (float64, error) {
	val := c.getData(key)
	if val != nil {
		return strconv.ParseFloat(val.(string), 64)
	}
	return 0.0, errors.New("not exist key:" + key)
}

// DefaultFloat returns the float64 value for a given key.
// if err != nil return defaltval
func (c *ZookeeperServiceConfiger) DefaultFloat(key string, defaultval float64) float64 {
	if v, err := c.Float(key); err == nil {
		return v
	}
	return defaultval
}

// String returns the string value for a given key.
func (c *ZookeeperServiceConfiger) String(key string) string {
	val := c.getData(key)
	if val != nil {
		if v, ok := val.(string); ok {
			return v
		}
	}
	return ""
}

// DefaultString returns the string value for a given key.
// if err != nil return defaltval
func (c *ZookeeperServiceConfiger) DefaultString(key string, defaultval string) string {
	// TODO FIXME should not use "" to replace non existence
	if v := c.String(key); v != "" {
		return v
	}
	return defaultval
}

// Strings returns the []string value for a given key.
func (c *ZookeeperServiceConfiger) Strings(key string) []string {
	stringVal := c.String(key)
	if stringVal == "" {
		return nil
	}
	return strings.Split(c.String(key), ";")
}

// DefaultStrings returns the []string value for a given key.
// if err != nil return defaltval
func (c *ZookeeperServiceConfiger) DefaultStrings(key string, defaultval []string) []string {
	if v := c.Strings(key); v != nil {
		return v
	}
	return defaultval
}

// DIY returns the raw value by a given key.
func (c *ZookeeperServiceConfiger) DIY(key string) (v interface{}, err error) {
	v = c.getData(key)
	if v != nil {
		return v, nil
	}
	return nil, errors.New("key  '" + key + "' not exist")
}

func (c *ZookeeperServiceConfiger) Data() map[string]interface{} {
	return c.data
}

// Set writes a new value for key.
func (c *ZookeeperServiceConfiger) Set(key string, val interface{}) error {
	keys := strings.Split(strings.ToLower(key), ".")
	key_len := len(keys)
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	data := c.data
	for index, key := range keys[0:] {
		if index == key_len-1 {
			data[key] = val
		} else {
			if _, ok := data[key]; ok {
				data, ok = data[key].(map[string]interface{})
				if !ok {
					data = make(map[string]interface{})
				}
			} else {
				data[key] = make(map[string]interface{})
			}
		}
	}
	return nil
}

// key
func (c *ZookeeperServiceConfiger) getData(key string) interface{} {
	if len(key) == 0 {
		return ""
	}
	path := c.config["root_path"] + "/" + strings.Trim(strings.Replace(key, ".", "/", -1), "/")
	if c.cache.cacheTimeout[path].Before(time.Now()) {
		value := c.loadData(path)
		err := c.Set(key, value)
		if err != nil {
			// log
		}
		c.cache.Mutex.RLock()
		c.cache.cacheTimeout[path] = time.Now().Add(c.cache.cacheTime)
		c.cache.Mutex.RUnlock()
		return value
	}

	keys := strings.Split(strings.ToLower(key), ".")
	c.Mutex.RLock()
	defer c.Mutex.RUnlock()
	value := c.data
	for _, key := range keys[0:] {
		if v, ok := value[key]; ok {
			if value, ok = v.(map[string]interface{}); !ok {
				return nil
			}
		}
	}

	return value
}

func (c *ZookeeperServiceConfiger) SetConn(conn *zk.Conn) {
	c.conn = conn
}

// GetConn 获取zookeeper连接
func (c *ZookeeperServiceConfiger) GetConn() *zk.Conn {
	return c.conn
}

func (c *ZookeeperServiceConfiger) loadData(path string) interface{} {
	var err error
	var value []byte
	var childNodes []string
	watcher, ok := c.watcher.keyWatcher[path]
	if !ok || !watcher {
		var eventCh <-chan zk.Event
		childNodes, _, eventCh, err = c.conn.ChildrenW(path)
		c.watcher.Mutex.Lock()
		c.watcher.keyWatcher[path] = true
		c.watcher.Mutex.Unlock()
		go c.watch(eventCh)
	} else {
		childNodes, _, err = c.conn.Children(path)
	}
	if err != nil {
		c.errors <- err
		return nil
	} else {
		if len(childNodes) > 0 {
			nodeData := make(map[string]interface{}, len(childNodes))
			for _, childNode := range childNodes {
				nodeData[childNode] = c.loadData(path + "/" + childNode)
			}
			return nodeData
		} else {
			watcher, ok := c.watcher.keyWatcher[path]
			if !ok || !watcher {
				var eventCh <-chan zk.Event
				value, _, eventCh, err = c.conn.GetW(path)
				c.watcher.Mutex.Lock()
				c.watcher.keyWatcher[path] = true
				c.watcher.Mutex.Unlock()
				go c.watch(eventCh)
			} else {
				value, _, err = c.conn.Get(path)
			}
			if err != nil {
				c.errors <- err
				return nil
			}
			if value != nil {
				return string(value)
			}
		}
	}
	return nil
}

func (c *ZookeeperServiceConfiger) watch(ch <-chan zk.Event) {
	for {
		select {
		case e := <-ch:
			if e.Err != nil {
				c.errors <- e.Err
			}
			switch e.Type {
			case zk.EventNodeDataChanged, zk.EventNodeCreated, zk.EventNodeDeleted, zk.EventNodeChildrenChanged:
				c.watcher.Mutex.Lock()
				c.watcher.keyWatcher[e.Path] = false
				c.watcher.Mutex.Unlock()
				c.cache.Mutex.RLock()
				c.cache.cacheTimeout[e.Path] = time.Now().Add(c.cache.cacheTime)
				c.cache.Mutex.RUnlock()
			}
			return
		}
	}
}

func (c *ZookeeperServiceConfiger) connect(config map[string]string) (err error) {
	if config["host"] == "" {
		return errConfigMissHost
	}
	var timeout int64
	if _, ok := config["timeout"]; !ok {
		timeout = 120
	} else {
		timeout, err = strconv.ParseInt(config["timeout"], 10, 64)
		if err != nil {
			panic(err)
		}
	}
	hosts := strings.Split(config["host"], ",")
	for index, host := range hosts {
		hosts[index] = strings.TrimSpace(host)
	}
	if c.conn, _, err = zk.Connect(hosts, time.Second*time.Duration(timeout)); err != nil {
		return err
	}
	if err = c.addAuth(config); err != nil {
		return err
	}
	return nil
}

func (c *ZookeeperServiceConfiger) addAuth(config map[string]string) (err error) {
	if config["user"] == "" {
		return errConfigMissUser
	}
	if config["password"] == "" {
		return errConfigMissPassword
	}
	if err = c.conn.AddAuth("digest", []byte(config["user"]+":"+config["password"])); err != nil {
		return err
	}
	return nil
}

func init() {
	config.RegisterService("zookeeper", &ZookeeperServiceConfig{})
}
