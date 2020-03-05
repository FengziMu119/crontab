package worker

import (
	"github.com/FengziMu119/crontab/common"
	"math/rand"
	"os/exec"
	"time"
)

//任务执行器
type Executor struct {
}

var (
	G_executor *Executor
)

// 执行一个任务
func (executor *Executor) ExecuteJob(info *common.JobExecuteInfo) {
	go func() {
		var (
			cmd     *exec.Cmd
			err     error
			outPut  []byte
			result  *common.JobExecuteResult
			jobLock *JobLock
		)
		//任务结果
		result = &common.JobExecuteResult{
			ExecuteInfo: info,
			Output:      make([]byte, 0),
		}
		//
		jobLock = G_jobMgr.CreateJobLock(info.Job.Name)
		// 上锁
		// 随机睡眠
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

		err = jobLock.TryLock()
		defer jobLock.Unlock()

		//记录任务开始时间
		result.StartTime = time.Now()
		//首先获取分布式锁
		if err != nil { //上锁失败
			result.Err = err
			result.EndTime = time.Now()
		} else {
			//上锁成功后，重置任务开启时间
			result.StartTime = time.Now()
			// 执行shell命令
			cmd = exec.CommandContext(info.CancleCtx, "/bin/bash", "-c", info.Job.Command)
			//执行并捕获输出
			outPut, err = cmd.CombinedOutput()
			// 记录任务结束时间
			result.EndTime = time.Now()
			result.Output = outPut
			result.Err = err

		}
		//任务执行完成后，把执行的结果返回给Scheduler,Scheduler会从executingTable中删除掉执行任务记录
		G_scheduler.PushJobResult(result)

	}()
}

//
func InitExecutor() (err error) {
	G_executor = &Executor{}
	return
}
