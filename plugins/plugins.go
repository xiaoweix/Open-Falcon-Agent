// Copyright 2017 Xiaomi, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plugins

type Plugin struct {
	FilePath string // 文件目录
	MTime    int64  //文件修改时间
	Cycle    int	// 监控周期
}

var (
	Plugins              = make(map[string]*Plugin) // 保存用户在 web 配置的要执行的 plugin，键值是 plugin 脚本目录
	PluginsWithScheduler = make(map[string]*PluginScheduler) // 保存定时执行的 plugin，键值是 plugin 脚本目录
)

//停止不再使用的 plugin，并从缓存的 plugins 表中删除
func DelNoUsePlugins(newPlugins map[string]*Plugin) {
	for currKey, currPlugin := range Plugins {
		newPlugin, ok := newPlugins[currKey]
		// 如果从 hbs 拉取的新 plugins 列表里没有该 plugin，或文件修改时间不同，删除老 plugin
		if !ok || currPlugin.MTime != newPlugin.MTime {
			deletePlugin(currKey)
		}
	}
}

//在缓存的 plugins 表中增加新 plugin，并启动定时执行 plugin
func AddNewPlugins(newPlugins map[string]*Plugin) {
	for fpath, newPlugin := range newPlugins {
		// 如果该 plugin 已经存在，且文件修改时间符合，跳过添加
		if _, ok := Plugins[fpath]; ok && newPlugin.MTime == Plugins[fpath].MTime {
			continue
		}

		Plugins[fpath] = newPlugin
		sch := NewPluginScheduler(newPlugin) // 生成定时 plugin
		PluginsWithScheduler[fpath] = sch
		sch.Schedule() // 启动定时执行
	}
}

func ClearAllPlugins() {
	for k := range Plugins {
		deletePlugin(k)
	}
}

func deletePlugin(key string) {
	v, ok := PluginsWithScheduler[key]
	if ok {
		v.Stop()
		delete(PluginsWithScheduler, key)
	}
	delete(Plugins, key)
}
