package master

import (
	"context"
	"github.com/FengziMu119/crontab/common"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type LogMgr struct {
	client        *mongo.Client
	logCollection *mongo.Collection
}

var (
	G_logMgr *LogMgr
)

func InitLogMgr() (err error) {
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
	G_logMgr = &LogMgr{
		client:        client,
		logCollection: client.Database("cron").Collection("log"),
	}
	return
}

func (logMgr *LogMgr) ListLog(name string, skip int, limit int) (logArr []*common.JobLog, err error) {
	var (
		filter  *common.JobLogFile
		cursor  *mongo.Cursor
		jobLog  *common.JobLog
		logSort *common.SortLogByStartTime
	)
	// 过滤条件
	filter = &common.JobLogFile{
		JobName: name,
	}
	logSort = &common.SortLogByStartTime{SortOrder: -1}
	findOptions := options.Find()
	findOptions.SetLimit(int64(limit))
	findOptions.SetSkip(int64(skip))
	findOptions.SetSort(logSort)
	//// 按照任务开始时间倒序
	logArr = make([]*common.JobLog, 0)
	//查询
	if cursor, err = logMgr.logCollection.Find(context.TODO(), filter, findOptions); err != nil {
		return
	}
	defer cursor.Close(context.TODO())
	for cursor.Next(context.TODO()) {
		jobLog = &common.JobLog{}
		//反序列化BSON
		if err = cursor.Decode(jobLog); err != nil {
			continue
		}
		logArr = append(logArr, jobLog)
	}
	return
}
