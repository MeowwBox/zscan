package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"zscan/config"
)

// var num int
type Service func(user string, pass string, addr string) (error, bool, string)

type burp_info struct {
	username string
	password string
	addr     string
}

type Burp struct {
	password_ch   chan string
	username_list []string
	username      string
	password      string
	userdict      string
	passdict      string
	aliveaddr     string
	tasklist      chan *burp_info
	service       Service
	wg            sync.WaitGroup
	burpthread    int
	stop          chan int8
	burpresult    string
}

func NewBurp(pass, user, userdict, passdict string, aliveaddr string, service Service, burpthread int) *Burp {
	return &Burp{password_ch: make(chan string, Thread*2),
		username_list: []string{},
		userdict:      userdict,
		passdict:      passdict,
		tasklist:      make(chan *burp_info, Thread*2),
		service:       service,
		aliveaddr:     aliveaddr,
		password:      pass,
		username:      user,
		burpthread:    burpthread,
		stop:          make(chan int8),
	}
}

func (b *Burp) Run() string {
	switch {
	case b.username != "" && b.userdict == "":
		if strings.Contains(b.username, ",") {
			userlist := strings.Split(b.username, ",")
			for _, user := range userlist {
				b.username_list = append(b.username_list, user)
			}
		} else {
			b.username_list = append(b.username_list, b.username)
		}
	case b.userdict != "":
		b.Getuser()
	default:
		b.username_list = []string{""}
	}
	switch {
	case b.password == "" && b.passdict != "":
		b.wg.Add(1)
		go b.Getpass()
	case b.password != "" && b.passdict == "":
		if strings.Contains(b.password, ",") {
			passlist := strings.Split(b.password, ",")
			for _, pass := range passlist {
				b.password_ch <- pass
			}
			close(b.password_ch)
		} else {
			b.password_ch <- b.password
			close(b.password_ch)
		}
	default:
		b.password_ch = make(chan string, len(config.Pass_dict))
		for _, i := range config.Pass_dict {
			b.password_ch <- i
		}
		close(b.password_ch)
	}
	b.wg.Add(1)
	go b.Gettasklist()
	for i := 0; i < b.burpthread; i++ {
		b.wg.Add(1)
		go b.Check()
	}
	b.wg.Wait()
	return b.burpresult
}

// 读取密码到缓冲信道中
func (b *Burp) Getpass() {
	b.readdict_To_Ch(b.passdict, &b.password_ch)
}

// 读取用户名到列表中
func (b *Burp) Getuser() {
	file, err := os.Open(b.userdict)
	Checkerr_exit(err)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		user := strings.TrimSpace(scanner.Text())
		if user != "" {

			b.username_list = append(b.username_list, user)
		}
	}
}

// 根据用户名密码还有生成任务列表
func (b *Burp) Gettasklist() {
	defer b.wg.Done()
	for pass := range b.password_ch {
		for _, user := range b.username_list {
			if cancelled(b.stop) {
				break
			}
			b.tasklist <- &burp_info{user, pass, b.aliveaddr}
		}
	}
	close(b.tasklist)
}

type burpresult struct {
	err        error
	success    bool
	servername string
}

func (b *Burp) Check() {
	defer b.wg.Done()
	for task := range b.tasklist {
		if cancelled(b.stop) {
			break
		}
		//if Verbose{
		//	fmt.Println(Yellow(fmt.Sprintf("Test:%v %v %v",task.addr,task.username,task.password)))
		//}

		timeoutch := make(chan burpresult)
		go func() {
			err, success, servername := b.service(task.username, task.password, task.addr)
			timeoutch <- burpresult{err: err, success: success, servername: servername}
		}()
		var err error
		var success bool
		var servername string
		select {
		case re := <-timeoutch:
			err = re.err
			success = re.success
			servername = re.servername
		case <-time.After(Timeout * 3):
			continue
		}
		//num+=1
		if err == nil && success {
			if cancelled(b.stop) {
				break
			} else {
				Output(fmt.Sprintf("\r[+]%v burp success:%v %v %v\n", servername, task.addr, task.username, task.password), LightGreen)
				if cancelled(b.stop) {
					break
				}
				b.burpresult = fmt.Sprintf("%v\tUsername:%v\tPassword:%v", servername, task.username, task.password)
				close(b.stop)
				break
			}
		}
	}
}

func (b *Burp) readdict_To_Ch(path string, ch *chan string) {
	defer b.wg.Done()
	defer close(*ch)
	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()
	f := bufio.NewReader(file)
	for {
		bur, err := f.ReadString('\n')
		if err != nil {
			//fmt.Println(Red(err))
			break
		}
		bur = strings.TrimSpace(bur)
		if cancelled(b.stop) {
			break
		}
		*ch <- bur
	}
}

func cancelled(stop chan int8) bool {
	select {
	case <-stop:
		return true
	default:
		return false
	}
}

//func Start_Burp(aliveip []string,check Service,t int)  {
//	var wg sync.WaitGroup
//	for _,i:=range aliveip{
//		wg.Add(1)
//		go func(i string) {
//			startburp:=NewBurp(Password,Username,Userdict,Passdict,i,check,t)
//			startburp.Run()
//			wg.Done()
//		}(i)
//	}
//	wg.Wait()
//}

//func bar()  {
//	for  {
//		for _, r := range `-\|/` {
//			fmt.Printf("\r%cAlready test:%v times %c", r,num,r)
//			time.Sleep(200 * time.Millisecond)
//		}
//	}
//}
