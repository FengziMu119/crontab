package worker

import (
	"context"
	"github.com/FengziMu119/crontab/common"
	"go.etcd.io/etcd/clientv3"
)

// 分布式锁(TXN事务)
type JobLock struct {
	kv         clientv3.KV
	lease      clientv3.Lease
	jobName    string //任务名
	cancelFunc context.CancelFunc
	leaseId    clientv3.LeaseID // 租约id
	isLock     bool             //是否上锁成功
}

// 初始化一把锁
func InitJobLock(jobName string, kv clientv3.KV, lease clientv3.Lease) (jobLock *JobLock) {
	jobLock = &JobLock{
		kv:      kv,
		lease:   lease,
		jobName: jobName,
	}
	return
}

//尝试上锁
func (jobLock *JobLock) TryLock() (err error) {
	var (
		leaseGrantResp *clientv3.LeaseGrantResponse
		cancelCtx      context.Context
		cancelFunc     context.CancelFunc
		leaseId        clientv3.LeaseID
		keepRespChan   <-chan *clientv3.LeaseKeepAliveResponse
		txn            clientv3.Txn
		lockKey        string
		txnResp        *clientv3.TxnResponse
	)
	//创建租约
	if leaseGrantResp, err = jobLock.lease.Grant(context.TODO(), 5); err != nil {
		return
	}
	//context用于取消自动续租
	cancelCtx, cancelFunc = context.WithCancel(context.TODO())
	//租约ID
	leaseId = leaseGrantResp.ID

	// 自动续费
	if keepRespChan, err = jobLock.lease.KeepAlive(cancelCtx, leaseId); err != nil {
		goto FALL
	}
	// 处理续租应答的协程
	go func() {
		var (
			keepResp *clientv3.LeaseKeepAliveResponse
		)
		for {
			select {
			case keepResp = <-keepRespChan: //自动续租应答
				if keepResp == nil {
					goto END
				}
			}
		}
	END:
	}()
	//创建事务
	txn = jobLock.kv.Txn(context.TODO())
	// 锁路径
	lockKey = common.JOB_LOCK_DIR + jobLock.jobName
	//事务抢锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseId))).
		Else(clientv3.OpGet(lockKey))
	//提交事务
	if txnResp, err = txn.Commit(); err != nil {
		goto FALL
	}
	//成功返回 失败释放租约
	if !txnResp.Succeeded { //锁被占用
		err = common.ERR_LOCK_ALREDY_REQUIRED
		goto FALL
	}
	//抢锁成功
	jobLock.leaseId = leaseId
	jobLock.cancelFunc = cancelFunc
	jobLock.isLock = true
	return
FALL:
	cancelFunc()                                  //取消自动续约
	jobLock.lease.Revoke(context.TODO(), leaseId) // 释放租约

	return
}

//释放锁
func (jobLock *JobLock) Unlock() {
	if jobLock.isLock {
		jobLock.cancelFunc() //取消我们程序自动续租协程
		jobLock.lease.Revoke(context.TODO(), jobLock.leaseId)
	}
}
