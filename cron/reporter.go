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
	"fmt"
	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/xiaoweix/Open-Falcon-Agent/g"
	"log"
	"time"
)

//向 hbs 汇报客户端状态
func ReportAgentStatus() {
	if g.Config().Heartbeat.Enabled && g.Config().Heartbeat.Addr != "" {
		go reportAgentStatus(time.Duration(g.Config().Heartbeat.Interval) * time.Second)
	}
}

//定时汇报客户端状态
func reportAgentStatus(interval time.Duration) {
	for {
		hostname, err := g.Hostname() // 获取当前主机的 hostname，依次尝试从配置文件、环境变量 FALCON_ENDPOINT、系统 hostname 获取
		if err != nil {
			hostname = fmt.Sprintf("error:%s", err.Error())
		}

		req := model.AgentReportRequest{
			Hostname:      hostname,
			IP:            g.IP(), // IP
			AgentVersion:  g.VERSION, // 版本
			PluginVersion: g.GetCurrPluginVersion(), // 插件版本号，即最后一次 git commit 的 hash
		}

		var resp model.SimpleRpcResponse
		err = g.HbsClient.Call("Agent.ReportStatus", req, &resp) // rpc 调用 hbs 的 Agent.ReportStatus，获取响应
		if err != nil || resp.Code != 0 {
			log.Println("call Agent.ReportStatus fail:", err, "Request:", req, "Response:", resp)
		}

		time.Sleep(interval) // 睡一个心跳汇报间隔时间
	}
}
