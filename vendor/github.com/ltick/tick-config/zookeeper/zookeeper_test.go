// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zookeeper

import (
	//"fmt"
	"testing"
)

func TestLoad(t *testing.T) {
	zk := new(ZookeeperServiceConfig)
	zkConf, err := zk.Load(map[string]string{
		"host":      "jx-op-zk00.zeus.lianjia.com,jx-op-zk01.zeus.lianjia.com",
		"user":      "platrd",
		"password":  "eiQuai8aes",
		"timeout":   "120",
		"root_path": "/platrd/nebula",
	})
	if err != nil {
		t.Fatal(err)
	}

	if zkConf.String("accounts.test.credential") != "LV56XXCKXJ4VW7X4K2GA:ZF04UHDP7erVckdkZWNlZ6bFwu2mte3dNi9aFhz7" {
		t.Fatal("get account error")
	}
}

func TestSet(t *testing.T) {
	zk := new(ZookeeperServiceConfig)
	zkConf, err := zk.Load(map[string]string{
		"host":      "jx-op-zk00.zeus.lianjia.com, jx-op-zk01.zeus.lianjia.com",
		"user":      "platrd",
		"password":  "eiQuai8aes",
		"timeout":   "120",
		"root_path": "/platrd/nebula",
	})
	if err != nil {
		t.Fatal(err)
	}

	zkConf.Set("accounts.test.credential", "test")
	if zkConf.String("accounts.test.credential") != "test" {
		t.Fatal("set account error")
	}
}

func TestSave(t *testing.T) {
	zk := new(ZookeeperServiceConfig)
	zkConf, err := zk.Load(map[string]string{
		"host":      "jx-op-zk00.zeus.lianjia.com, jx-op-zk01.zeus.lianjia.com",
		"user":      "platrd",
		"password":  "eiQuai8aes",
		"timeout":   "120",
		"root_path": "/platrd/nebula",
	})
	if err != nil {
		t.Fatal(err)
	}

	zkConf.SaveConfigFile("credential.ini")
}
