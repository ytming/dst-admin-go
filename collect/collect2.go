package collect

import (
	"dst-admin-go/entity"
	"fmt"
	"github.com/hpcloud/tail"
	"log"
	"path/filepath"
	"strings"
	"time"
)

type Collect struct {
	state             chan int
	stop              chan bool
	severLogList      []string
	serverChatLogList []string
	length            int
}

func NewCollect(baseLogPath string) *Collect {

	collect := &Collect{
		state: make(chan int, 1),
		stop:  make(chan bool, 4),
		severLogList: []string{
			filepath.Join(baseLogPath, "Master", "server_log.txt"),
			filepath.Join(baseLogPath, "Caves", "server_log.txt"),
		},
		serverChatLogList: []string{
			filepath.Join(baseLogPath, "Master", "server_chat_log.txt"),
			filepath.Join(baseLogPath, "Master", "server_chat_log.txt"),
		},
		length: 4,
	}
	collect.state <- 1
	return collect
}

func (c *Collect) ReCollect() {
	for i := 0; i < c.length; i++ {
		c.stop <- true
	}
	c.state <- 1
}

func (c *Collect) StartCollect() {
	for {
		select {
		case <-c.state:
			// 采集
			for _, s := range c.severLogList {
				go c.tailServeLog(s)
			}
			for _, s := range c.serverChatLogList {
				go c.tailServerChatLog(s)
			}
		default:
			time.Sleep(5 * time.Second)
			continue
		}
	}
}

func (c *Collect) parseSpawnRequestLog(text string) {
	// Spawn request: wurt from 猜猜我是谁
	arr := strings.Split(text, " ")
	temp := strings.Replace(arr[0], " ", "", -1)
	t := temp[:len(temp)-1]
	role := strings.Replace(arr[3], " ", "", -1)
	name := strings.Replace(arr[5], "\n", "", -1)

	spawn := entity.Spawn{Name: name, Role: role, Time: t}
	entity.DB.Create(&spawn)
}

func (c *Collect) parseRegenerateLog(text string) {

}

func (c *Collect) parseNewIncomingLog(lines []string) {

	connect := entity.Connect{}
	for i, line := range lines {
		fmt.Println(i, line)
		if i == 1 {
			// 解析 ip
			str := strings.Split(line, " ")
			if len(str) < 5 {
				log.Println("[EROOR] str 解析错误: ", str)
				connect.Ip = ""
			} else {
				var ip string
				if strings.Contains(line, "[LAN]") {
					ip = str[5]
				} else {
					ip = str[4]
				}
				connect.Ip = ip
				fmt.Println("ip", ip)
			}
		}
		if i == 2 {
			// 解析 ip
		}
		if i == 3 {
			// 解析 KuId 和 用户名
			str := strings.Split(line, " ")
			if len(str) <= 4 {
				log.Println("[EROOR] str 解析错误: ", str)
			} else {
				ku := str[3]
				ku = ku[1 : len(ku)-1]
				name := str[4]
				connect.Name = name
				connect.KuId = ku
				fmt.Println("ku", ku, "name", name)
			}
		}
		if i == 4 {
			// 解析 steamId
			str := strings.Split(line, " ")
			if len(str) < 4 {
				log.Println("[EROOR] str 解析错误: ", str)
			} else {
				steamId := str[4]
				steamId = steamId[1 : len(steamId)-1]
				fmt.Println("steamId", steamId)
				connect.SteamId = steamId
			}
		}
	}
	entity.DB.Create(&connect)
}

func (c *Collect) tailServeLog(fileName string) {

	log.Println("开始采集 path:", fileName)
	config := tail.Config{
		ReOpen:    true,                                 // 重新打开
		Follow:    true,                                 // 是否跟随
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2}, // 从文件的哪个地方开始读
		MustExist: false,                                // 文件不存在不报错
		Poll:      true,
	}
	tails, err := tail.TailFile(fileName, config)
	if err != nil {
		log.Println("文件监听失败", err)
	}
	var (
		which        = 0
		isNewConnect = false
		incoming     = make([]string, 4)
	)
	for {
		select {
		case line, ok := <-tails.Lines:
			if !ok {
				log.Println("文件读取失败", err)
				time.Sleep(time.Second)
			} else {
				text := line.Text
				if find := strings.Contains(text, "Spawn request"); find {
					c.parseSpawnRequestLog(text)
				} else if find := strings.Contains(text, "regenerate"); find {
					c.parseSpawnRequestLog(text)
				} else if find := strings.Contains(text, "New incoming connection"); find {
					isNewConnect = true
				}
				// 获取接下来的五条数据
				if isNewConnect {
					incoming = append(incoming, text)
					which++
					if which > 4 {
						isNewConnect = false
						incoming = make([]string, 4)
						c.parseNewIncomingLog(incoming)
					}
				}
			}
		case <-c.stop:
			// 结束监听
			err := tails.Stop()
			if err != nil {
				log.Println("tail log 结束失败")
				return
			}
			return
		}
	}
}

func (c *Collect) parseChatLog(text string) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("玩家行为日志解析异常:", err)
		}
	}()
	//[00:00:55]: [Join Announcement] 猜猜我是谁
	if strings.Contains(text, "[Join Announcement]") {
		c.parseJoin(text)
	}
	//[00:02:28]: [Leave Announcement] 猜猜我是谁
	if strings.Contains(text, "[Leave Announcement]") {
		c.parseLeave(text)
	}
	//[00:02:17]: [Death Announcement] 猜猜我是谁 死于： 采摘的红蘑菇。她变成了可怕的鬼魂！
	if strings.Contains(text, "[Death Announcement]") {
		c.parseDeath(text)
	}
	//[00:02:37]: [Resurrect Announcement] 猜猜我是谁 复活自： TMIP 控制台.
	if strings.Contains(text, "[Resurrect Announcement]") {
		c.parseResurrect(text)
	}
	//[00:03:16]: [Say] (KU_Mt-zrX8K) 猜猜我是谁: 你好啊
	if strings.Contains(text, "[Say]") {
		c.parseSay(text)
	}
}

func (c *Collect) parseSay(text string) {
	fmt.Println(text)

	arr := strings.Split(text, " ")
	temp := strings.Replace(arr[0], " ", "", -1)
	t := temp[:len(temp)-1]
	action := arr[1]
	kuId := arr[2]
	kuId = kuId[1 : len(kuId)-1]
	name := arr[3]
	name = name[:len(name)-1]
	rest := ""
	for i := 4; i <= len(arr)-1; i++ {
		rest += arr[i] + " "
	}
	actionDesc := rest

	spawn := c.getSpawnRole(name)
	connect := c.getConnectInfo(name)

	playerLog := entity.PlayerLog{
		Name:       name,
		Role:       spawn.Role,
		Action:     action,
		ActionDesc: actionDesc,
		Time:       t,
		Ip:         connect.Ip,
		KuId:       kuId,
		SteamId:    connect.SteamId,
	}
	entity.DB.Create(&playerLog)
}

func (c *Collect) parseResurrect(text string) {
	c.parseDeath(text)
}

func (c *Collect) parseDeath(text string) {
	fmt.Println(text)
	arr := strings.Split(text, " ")

	temp := strings.Replace(arr[0], " ", "", -1)
	t := temp[:len(temp)-1]
	action := arr[1] + arr[2]
	name := strings.Replace(arr[3], "\n", "", -1)

	rest := ""
	for i := 4; i <= len(arr)-1; i++ {
		rest += arr[i] + " "
	}
	actionDesc := rest

	//获取最近的一条spwan记录和newComing
	spawn := c.getSpawnRole(name)
	connect := c.getConnectInfo(name)
	fmt.Println(connect)

	playerLog := entity.PlayerLog{
		Name:       name,
		Role:       spawn.Role,
		Action:     action,
		ActionDesc: actionDesc,
		Time:       t,
		Ip:         connect.Ip,
		KuId:       connect.KuId,
		SteamId:    connect.SteamId,
	}

	entity.DB.Create(&playerLog)

}

func (c *Collect) parseLeave(text string) {
	c.parseJoin(text)
}

func (c *Collect) parseJoin(text string) {
	fmt.Println(text)
	arr := strings.Split(text, " ")
	temp := strings.Replace(arr[0], " ", "", -1)
	t := temp[:len(temp)-1]
	action := arr[1] + arr[2]
	name := arr[3]

	spawn := c.getSpawnRole(name)
	connect := c.getConnectInfo(name)

	playerLog := entity.PlayerLog{
		Name:    name,
		Role:    spawn.Role,
		Action:  action,
		Time:    t,
		Ip:      connect.Ip,
		KuId:    connect.KuId,
		SteamId: connect.SteamId,
	}
	entity.DB.Create(&playerLog)
}

func (c *Collect) tailServerChatLog(fileName string) {
	log.Println("开始采集 path:", fileName)
	config := tail.Config{
		ReOpen:    true,                                 // 重新打开
		Follow:    true,                                 // 是否跟随
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2}, // 从文件的哪个地方开始读
		MustExist: false,                                // 文件不存在不报错
		Poll:      true,
	}
	tails, err := tail.TailFile(fileName, config)
	if err != nil {
		log.Println("文件监听失败", err)
	}
	for {
		select {
		case line, ok := <-tails.Lines:
			if !ok {
				log.Println("文件读取失败", err)
				time.Sleep(time.Second)
			} else {
				text := line.Text
				c.parseChatLog(text)
			}
		case <-c.stop:
			// 结束监听
			err := tails.Stop()
			if err != nil {
				log.Println("tail log 结束失败")
				return
			}
			return
		}
	}
}

func (c *Collect) getSpawnRole(name string) *entity.Spawn {
	spawn := new(entity.Spawn)
	entity.DB.Where("name LIKE ?", "%"+name+"%").Last(spawn)
	return spawn
}

func (c *Collect) getConnectInfo(name string) *entity.Connect {
	connect := new(entity.Connect)
	entity.DB.Where("name LIKE ?", "%"+name+"%").Last(connect)
	return connect
}
