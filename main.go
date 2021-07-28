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

package main

import (
	"flag"
	"fmt"
	"github.com/xiaoweix/Open-Falcon-Agent/cron"
	"github.com/xiaoweix/Open-Falcon-Agent/funcs"
	"github.com/xiaoweix/Open-Falcon-Agent/g"
	"github.com/xiaoweix/Open-Falcon-Agent/http"

	"os"
)

func main() {

	// 命令行参数
	cfg := flag.String("c", "cfg.json", "configuration file")
	version := flag.Bool("v", false, "show version")
	check := flag.Bool("check", false, "check collector")

	// 获取一下命令行参数
	flag.Parse()

	// 打印版本
	if *version {
		fmt.Println(g.VERSION)
		os.Exit(0)
	}

	// 检查是否能正常采集监控数据
	if *check {
		funcs.CheckCollector()
		os.Exit(0)
	}

	// 读取配置文件
	g.ParseConfig(*cfg)

	// 设置日志级别
	if g.Config().Debug {
		g.InitLog("debug")
	} else {
		g.InitLog("info")
	}

	// 初始化
	g.InitRootDir() // 初始化根目录
	g.InitLocalIp() // 初始化本地 IP
	g.InitRpcClients() // 初始化 hbs rpc 客户端

	funcs.BuildMappers() // 建立收集数据的函数和发送数据间隔映射

	// 定时任务（每秒），通过文件 /proc/cpustats 和 /proc/diskstats，更新 cpu 和 disk 的原始数据，等待后一步处理
	go cron.InitDataHistory()

	// 定时任务
	cron.ReportAgentStatus() // 向 hbs 发送客户端心跳信息
	cron.SyncMinePlugins() // 同步 plugins
	cron.SyncBuiltinMetrics() // 从 hbs 同步 BuiltinMetrics
	cron.SyncTrustableIps() // 同步可信任 IP 地址

	cron.Collect()  // 采集监控数据
	//重点了，其实里面这些采集函数都大同小异，都是通过第三方库nux读取一些系统文件，再做一些计算百分比之类的简单处理
	//collector.go只是一个壳，定义了一些封装的逻辑，采集数据的函数都保存在funcs.go里面的Mapper里

	go http.Start() // 开启 http 服务

	select {} // 阻塞 main 函数

}
