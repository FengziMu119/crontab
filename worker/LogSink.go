package worker

import (
	"context"
	"github.com/FengziMu119/crontab/common"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

//MongoDB存储日志
type LogSink struct {
	client         *mongo.Client
	logCollection  *mongo.Collection
	logChan        chan *common.JobLog
	autoCommitChan chan *common.LogBatch
}

var (
	G_logSink *LogSink
)

// 批量写入日志
func (logSink *LogSink) saveLogs(batch *common.LogBatch) {
	logSink.logCollection.InsertMany(context.TODO(), batch.Logs)
}

func (logSink *LogSink) writeLoop() {
	var (
		log          *common.JobLog
		logBatch     *common.LogBatch //当前批次
		commitTimer  *time.Timer
		timeoutBatch *common.LogBatch //超时批次
	)
	for {
		select {
		case log = <-logSink.logChan:
			if logBatch == nil {
				logBatch = &common.LogBatch{}
			}
			commitTimer = time.AfterFunc(
				time.Duration(G_config.JobLogCommitTimeOut)*time.Millisecond,
				func(batch *common.LogBatch) func() {
					return func() {
						logSink.autoCommitChan <- batch
					}
				}(logBatch),
			)
			//把新的日志放到数组
			logBatch.Logs = append(logBatch.Logs, log)
			// 如果批次满了 立即发送
			if len(logBatch.Logs) >= G_config.JobLogBatchSize {
				//发送日志
				logSink.saveLogs(logBatch)
				//清空logBatch
				logBatch = nil
				//取消定时器
				commitTimer.Stop()
			}
		case timeoutBatch = <-logSink.autoCommitChan: //过期的批次
			//判断过期批次是否仍然是当前批次
			if timeoutBatch != logBatch {
				continue
			}
			// 把批次写入到MongoDB中
			logSink.saveLogs(timeoutBatch)
			//清空批次
			logBatch = nil

		}
	}
}

func InitLogSink() (err error) {
	var (
		client        *mongo.Client
		clientOptions *options.ClientOptions
		ctx           context.Context
	)
	ctx, _ = context.WithTimeout(context.Background(), time.Duration(G_config.MongodbConnectTimeOut)*time.Millisecond)
	clientOptions = options.Client().ApplyURI(G_config.MongoUri)
	if client, err = mongo.Connect(ctx, clientOptions); err != nil {
		return
	}
	//选择DB和collection
	G_logSink = &LogSink{
		client:         client,
		logCollection:  client.Database("cron").Collection("log"),
		logChan:        make(chan *common.JobLog, 1000),
		autoCommitChan: make(chan *common.LogBatch, 1000),
	}
	//启动写日志协程
	go G_logSink.writeLoop()
	return
}

// 发送日志
func (logSink *LogSink) Append(jobLog *common.JobLog) {
	select {
	case logSink.logChan <- jobLog:
	default:

	}

}
