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

package cron

import (
	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/xiaoweix/Open-Falcon-Agent/g"
	"github.com/xiaoweix/Open-Falcon-Agent/plugins"
	"log"
	"strings"
	"time"
)
//同步用户自定义 plugins
func SyncMinePlugins() {

	// 配置未使能 plugin，返回
	if !g.Config().Plugin.Enabled {
		return
	}

	// 配置未使能 hbs，返回
	if !g.Config().Heartbeat.Enabled {
		return
	}

	// 配置未使能 hbs地址，返回
	if g.Config().Heartbeat.Addr == "" {
		return
	}

	go syncMinePlugins()
}

//同步 plugins
func syncMinePlugins() {

	var (
		timestamp  int64 = -1
		pluginDirs []string
	)

	duration := time.Duration(g.Config().Heartbeat.Interval) * time.Second

	for {
		time.Sleep(duration) // 睡一个周期间隔

		hostname, err := g.Hostname() // 获取当前主机的 hostname，依次尝试从配置文件、环境变量 FALCON_ENDPOINT、系统 hostname 获取
		if err != nil {
			continue
		}

		req := model.AgentHeartbeatRequest{ // 新建 rpc 请求
			Hostname: hostname,
		}

		var resp model.AgentPluginsResponse
		err = g.HbsClient.Call("Agent.MinePlugins", req, &resp) // rpc 调用 hbs 的 Agent.MinePlugins
		if err != nil {
			log.Println("ERROR:", err)
			continue
		}

		if resp.Timestamp <= timestamp { // 如果返回数据的时间戳小于等于上次的同步的时间戳，跳过本次同步
			continue
		}

		pluginDirs = resp.Plugins
		timestamp = resp.Timestamp

		if g.Config().Debug {
			log.Println(&resp)
		}

		if len(pluginDirs) == 0 { // 若用户没有配置，清空缓存的 plugins
			plugins.ClearAllPlugins()
		}

		desiredAll := make(map[string]*plugins.Plugin) // 保存配置的目录下所有符合命名规则的监控脚本

		for _, p := range pluginDirs {
			underOneDir := plugins.ListPlugins(strings.Trim(p, "/")) // 列出该目录下所有符合命名规则setp_xxx的监控脚本（用户配置的路径，前后的“/”会被去掉）
			for k, v := range underOneDir {
				desiredAll[k] = v
			}
		}

		// 更新 plugins 缓存
		plugins.DelNoUsePlugins(desiredAll) // 清除无用的
		plugins.AddNewPlugins(desiredAll) // 增加新的

	}
}
