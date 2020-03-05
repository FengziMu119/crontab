package master

import (
	"encoding/json"
	"fmt"
	"github.com/FengziMu119/crontab/common"
	"net"
	"net/http"
	"strconv"
	"time"
)

// 任务的HTTP接口
type ApiServer struct {
	httpServer *http.Server
}

var (
	// 单例对象
	G_aipServer *ApiServer
)

// 保存任务接口
// post job = {"name":"job1","command":"echo hello","cronExpr":"****"}
func handleJobSave(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		postJob string
		job     common.Job
		oldJob  *common.Job
		bytes   []byte
	)
	// 解析post表单
	if err = r.ParseForm(); err != nil {
		goto ERR
	}
	//取表单中的job字段
	postJob = r.PostForm.Get("job")
	fmt.Println(postJob)
	// 反序列化job
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		goto ERR
	}
	// 保存到ETCD中
	if oldJob, err = G_jobMgr.SaveJob(&job); err != nil {
		goto ERR
	}

	//返回正常应答
	if bytes, err = common.BulidResponse(0, "success", oldJob); err == nil {
		w.Write(bytes)
	}
	return

ERR:
	if bytes, err = common.BulidResponse(-1, err.Error(), nil); err == nil {
		w.Write(bytes)
	}
}

// 删除任务接口
func handleJobDelete(w http.ResponseWriter, r *http.Request) {
	var (
		err    error
		name   string
		oldJob *common.Job
		bytes  []byte
	)

	if err = r.ParseForm(); err != nil {
		goto ERR
	}
	// 删除的任务名
	name = r.PostForm.Get("name")

	// 去删除任务
	if oldJob, err = G_jobMgr.DeleteJob(name); err != nil {
		goto ERR
	}
	//返回正常应答
	if bytes, err = common.BulidResponse(0, "success", oldJob); err == nil {
		w.Write(bytes)
	}
	return

ERR:
	if bytes, err = common.BulidResponse(-1, err.Error(), nil); err == nil {
		w.Write(bytes)
	}
}

// 查询接口
func handleJobList(w http.ResponseWriter, r *http.Request) {
	var (
		jobList []*common.Job
		err     error
		bytes   []byte
	)
	if jobList, err = G_jobMgr.ListJobs(); err != nil {
		goto ERR
	}
	//返回正常应答
	if bytes, err = common.BulidResponse(0, "success", jobList); err == nil {
		w.Write(bytes)
	}
	return

ERR:
	if bytes, err = common.BulidResponse(-1, err.Error(), nil); err == nil {
		w.Write(bytes)
	}
}

//杀死任务
func handleJobKill(w http.ResponseWriter, r *http.Request) {
	var (
		err   error
		name  string
		bytes []byte
	)
	// 解析post表单
	if err = r.ParseForm(); err != nil {
		goto ERR
	}
	// 要杀死的任务名
	name = r.PostForm.Get("name")
	if err = G_jobMgr.KillJob(name); err != nil {
		goto ERR
	}
	if bytes, err = common.BulidResponse(0, "success", nil); err == nil {
		w.Write(bytes)
	}
	return

ERR:
	if bytes, err = common.BulidResponse(-1, err.Error(), nil); err == nil {
		w.Write(bytes)
	}

}

// 查询任务日志
func handleJobLog(w http.ResponseWriter, r *http.Request) {
	var (
		err        error
		name       string // 任务名字
		skipParam  string //从第几条开始
		limitParam string // 返回多少条
		skip       int
		limit      int
		logArr     []*common.JobLog
		bytes      []byte
	)

	// 解析GET参数
	if err = r.ParseForm(); err != nil {
		goto ERR

	}

	//获取请求参数
	name = r.Form.Get("name")
	skipParam = r.Form.Get("skip")
	limitParam = r.Form.Get("limit")
	if skip, err = strconv.Atoi(skipParam); err != nil {
		skip = 0
	}
	if limit, err = strconv.Atoi(limitParam); err != nil {
		limit = 20
	}
	if logArr, err = G_logMgr.ListLog(name, skip, limit); err != nil {
		goto ERR
	}
	if bytes, err = common.BulidResponse(0, "success", logArr); err == nil {
		w.Write(bytes)
	}
	return

ERR:
	if bytes, err = common.BulidResponse(-1, err.Error(), nil); err == nil {
		w.Write(bytes)
	}
}

func InitApiServer() (err error) {
	var (
		mux           *http.ServeMux
		listener      net.Listener
		httpServer    *http.Server
		staticDir     http.Dir
		staticHandler http.Handler
	)
	//配置路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)
	mux.HandleFunc("/job/delete", handleJobDelete)
	mux.HandleFunc("/job/list", handleJobList)
	mux.HandleFunc("/job/kill", handleJobKill)
	mux.HandleFunc("/job/log", handleJobLog)

	//静态文件目录
	staticDir = http.Dir(G_config.WebRoot)
	staticHandler = http.FileServer(staticDir)
	mux.Handle("/", http.StripPrefix("/", staticHandler))

	// 启动TCP监听
	if listener, err = net.Listen("tcp", ":"+strconv.Itoa(G_config.ApiPort)); err != nil {
		return
	}

	//创建一个HTTP服务
	httpServer = &http.Server{
		ReadTimeout:  time.Duration(G_config.ApiReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(G_config.ApiReadTimeout) * time.Millisecond,
		Handler:      mux,
	}
	// 赋值单例
	G_aipServer = &ApiServer{
		httpServer: httpServer,
	}
	// 启动服务端
	go httpServer.Serve(listener)
	return
}
