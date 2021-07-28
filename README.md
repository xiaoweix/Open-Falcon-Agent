# open-falcon agent源码阅读与解析
关于open-falcon的架构及说明可参考open-falcon[官方网站](http://open-falcon.org/)。 目前小米、美团、滴滴、360、金山云、新浪微博、京东、爱奇艺等都在使用open-falcon或者基于open-falcon的二次开发。open-falcon的架构清晰，代码难度不高，非常适合阅读学习。这里对open-falcon的源码进行解析。

falcon-agent是一个Linux的监控插件. 它很像是 zabbix-agent 和 tcollector.


## 如何安装

这个是一个golang语言开发的应用程序，以下是安装步骤

```bash
# set $GOPATH and $GOROOT
mkdir -p $GOPATH/src/github.com/open-falcon
cd $GOPATH/src/github.com/open-falcon
git clone https://github.com/open-falcon/falcon-plus.git
cd falcon-plus/modules/agent
go get
./control build
./control start

# goto http://localhost:1988
```

## 配置

- heartbeat: 心跳监测的 rpc 地址
- transfer: transfer rpc 的地址
- ignore: 需要忽略的监控指标

# 一、 open-falcon介绍

​    监控系统是整个运维环节，乃至整个产品生命周期中最重要的一环，事前及时预警发现故障，事后提供翔实的数据用于追查定位问题。监控系统作为一个成熟的运维产品，业界有很多开源的实现可供选择。当公司刚刚起步，业务规模较小，运维团队也刚刚建立的初期，选择一款开源的监控系统，是一个省时省力，效率最高的方案。之后，随着业务规模的持续快速增长，监控的对象也越来越多，越来越复杂，监控系统的使用对象也从最初少数的几个SRE，扩大为更多的DEVS，SRE。这时候，监控系统的容量和用户的“使用效率”成了最为突出的问题。

监控系统业界有很多杰出的开源监控系统。我们在早期，一直在用zabbix，不过随着业务的快速发展，以及互联网公司特有的一些需求，现有的开源的监控系统在性能、扩展性、和用户的使用效率方面，已经无法支撑了。

因此，我们在过去的一年里，从互联网公司的一些需求出发，从各位SRE、SA、DEVS的使用经验和反馈出发，结合业界的一些大的互联网公司做监控，用监控的一些思考出发，设计开发了小米的监控系统：open-falcon。

# 二、 open-falcon特点

1、强大灵活的数据采集：自动发现，支持falcon-agent、snmp、支持用户主动push、用户自定义插件支持、opentsdb data model like（timestamp、endpoint、metric、key-value tags）

2、水平扩展能力：支持每个周期上亿次的数据采集、告警判定、历史数据存储和查询

3、高效率的告警策略管理：高效的portal、支持策略模板、模板继承和覆盖、多种告警方式、支持callback调用

4、人性化的告警设置：最大告警次数、告警级别、告警恢复通知、告警暂停、不同时段不同阈值、支持维护周期

5、高效率的graph组件：单机支撑200万metric的上报、归档、存储（周期为1分钟）

6、高效的历史数据query组件：采用rrdtool的数据归档策略，秒级返回上百个metric一年的历史数据

7、dashboard：多维度的数据展示，用户自定义Screen

8、高可用：整个系统无核心单点，易运维，易部署，可水平扩展

9、开发语言： 整个系统的后端，全部golang编写，portal和dashboard使用python编写。

# 三、源码解析

## 1、agent介绍及源码解析

### 1）介绍

agent是open-falcon中核心的组件之一、它主要负责主机监控信息的收集。

每台服务器，都有安装falcon-agent，falcon-agent是一个golang开发的daemon程序，用于自发现的采集单机的各种数据和指标，这些指标包括不限于以下几个方面，共计200多项指标。

CPU相关

磁盘相关

IO

Load

内存相关

网络相关

端口存活、进程存活

ntp offset（插件）

某个进程资源消耗（插件）

netstat、ss 等相关统计项采集

机器内核配置参数

只要安装了falcon-agent的机器，就会自动开始采集各项指标，主动上报，不需要用户在server做任何配置（这和zabbix有很大的不同），这样做的好处，就是用户维护方便，覆盖率高。当然这样做也会server端造成较大的压力，不过open-falcon的服务端组件单机性能足够高，同时都可以水平扩展，所以自动多采集足够多的数据，反而是一件好事情，对于SRE和DEV来讲，事后追查问题，不再是难题。

另外，falcon-agent提供了一个proxy-gateway，用户可以方便的通过http接口，push数据到本机的gateway，gateway会帮忙高效率的转发到server端。

### 2）源码

源码位置

```
modules/agent
```

首先来看看main函数 程序的入口

```go
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
```

agent中最主要的是funcs包，这个包下放着最主要的CPU监控、内存监控、磁盘监控等所有逻辑。主要来看一下CPU监控的实现

```go
import (
	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/toolkits/nux"
	"sync"
)

// 保留多少轮历史数据
const (
	historyCount int = 2
)


var (
	procStatHistory [historyCount]*nux.ProcStat // 存放 cpu 历史数据
	psLock          = new(sync.RWMutex)  // 锁
)

//更新 cpu 状态
func UpdateCpuStat() error {

	// 读取 /proc/stat 文件 /proc/stat这个文件记录了开机以来的cpu使用信息
	ps, err := nux.CurrentProcStat()
	if err != nil {
		return err
	}

	psLock.Lock()
	defer psLock.Unlock()

	// 抛弃过期的历史数据，给新数据腾出位置
	for i := historyCount - 1; i > 0; i-- {
		procStatHistory[i] = procStatHistory[i-1]
	}

	procStatHistory[0] = ps // 保存最新数据
	return nil
}
```

其实对于监控大量使用了`github.com/toolkits/nux`这个包，想实现监控的可以学着使用这个包。

其他的包的源码注释已经写在了代码中了，可以直接查看。
