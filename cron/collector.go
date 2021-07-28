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
	"time"

	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/xiaoweix/Open-Falcon-Agent/funcs"
	"github.com/xiaoweix/Open-Falcon-Agent/g"
)
//定时（每秒）通过 /proc/stat 和 /proc/diskstats 文件采集 cpu 和 disk 的数据，保留最近两次的数据

func InitDataHistory() {
	for {
		funcs.UpdateCpuStat()  // 更新 cpu 状态
		funcs.UpdateDiskStats() // 更新 disk 状态
		time.Sleep(g.COLLECT_INTERVAL)  // 睡一个收集间隔（一秒）
	}
}
//采集数据
func Collect() {

	if !g.Config().Transfer.Enabled {
		return
	}

	if len(g.Config().Transfer.Addrs) == 0 {
		return
	}

	for _, v := range funcs.Mappers {
		go collect(int64(v.Interval), v.Fs)
	}
}
// 采集单组数据
func collect(sec int64, fns []func() []*model.MetricValue) {
	t := time.NewTicker(time.Second * time.Duration(sec)) //定时器
	defer t.Stop()
	for {
		<-t.C

		hostname, err := g.Hostname()
		if err != nil {
			continue
		}

		mvs := []*model.MetricValue{} // 保存采集后的 metric
		ignoreMetrics := g.Config().IgnoreMetrics // 配置的ignore metric

		for _, fn := range fns {
			items := fn()  // 调用函数，获取监控值
			if items == nil {
				continue
			}

			if len(items) == 0 {
				continue
			}

			for _, mv := range items {
				// 如果配置了忽略，跳过数据采集
				if b, ok := ignoreMetrics[mv.Metric]; ok && b {
					continue
				} else {
					mvs = append(mvs, mv)
				}
			}
		}
		// 设置时间戳、endpoint、step
		now := time.Now().Unix()
		for j := 0; j < len(mvs); j++ {
			mvs[j].Step = sec
			mvs[j].Endpoint = hostname
			mvs[j].Timestamp = now
		}

		// 发送给 transfer
		g.SendToTransfer(mvs)

	}
}
