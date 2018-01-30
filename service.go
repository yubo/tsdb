package tsdb

import (
	"fmt"
	"sync"

	"github.com/golang/glog"
)

type TsdbService struct {
	sync.RWMutex
	buckets map[int]*BucketMap
	ids     []int64
}

func NewService() *TsdbService {
	return nil
}

func (t *TsdbService) Start() (err error) {
	// config adapter not done
	t.ids = make([]int64, 1)
	t.ids[0] = 1

	t.buckets = make(map[int]*BucketMap)

	k := NewKeyListWriter(DataDirectory_Test, 100)
	b := NewBucketLogWriter(4*3600, DataDirectory_Test, 100, 0)

	for _, shardId := range t.ids {
		// todo
		if err := PathCreate(shardId); err != nil {
			return err
		}
		t.buckets[int(shardId)] = NewBucketMap(6, 4*3600, shardId, DataDirectory_Test,
			k, b, UNOWNED)
	}

	// check
	go func() {
		for _, m := range t.buckets {
			if err := m.SetState(PRE_OWNED); err != nil {
				glog.Fatal("set state failed")
			}
			if err := m.ReadKeyList(); err != nil {
				glog.Fatal(err)
			}
			if err := m.ReadData(); err != nil {
				glog.Fatal(err)
			}
		}
		for _, m := range t.buckets {
			more, err := m.ReadBlockFiles()
			if err != nil {
				glog.Fatal(err)
			}

			for more {
				more, err = m.ReadBlockFiles()
				if err != nil {
					glog.Fatal(err)
				}
			}
		}

	}()

	return nil
}

func (t *TsdbService) Put(req *PutRequest) (*PutResponse, error) {
	res := &PutResponse{}
	for _, data := range req.Data {
		m := t.buckets[int(data.Key.ShardId)]
		if m == nil {
			return res, fmt.Errorf("key not exit")
		}

		newRows, dataPoints, err := m.Put(data.Key.Key, TimeValuePair{Value: data.Value.Value,
			Timestamp: data.Value.Timestamp}, 0, false)
		if err != nil {
			return res, err
		}

		if newRows == NOT_OWNED && dataPoints == NOT_OWNED {
			return res, fmt.Errorf("key not own!")
		}

		res.N++
	}

	return res, nil
}

func (t *TsdbService) Get(req *GetRequest) (*GetResponse, error) {
	return nil, nil
	/*
		res := &GetResponse{}
		if lne(req.Key.Key) == 0 {
			return nil, fmt.Errorf("null key!")
		}

		m := t.buckets[int(req.ShardId)]
		if m == nil {
			return nil, fmt.Errorf("key not exit")
		}

		res.Key = req.Key
		state := m.GetState()
		switch state {
		case UNOWNED:
			return nil, fmt.Errorf("Don't own shard %d", req.ShardId)
		case PRE_OWNED, READING_KEYS, READING_KEYS_DONE, READING_LOGS, PROCESSING_QUEUED_DATA_POINTS:
			return nil, fmt.Errorf("Shard %d in progress", req.ShardId)
		default:
			datas, err := m.Get(req.Key, req.Begin, req.End)
			if err != nil {
				return res, err
			}

			for _, dp := range datas {
				res.Dps = append(res.Dps, &DataPoint{Value: dp.Value, Timestamp: dp.Timestamp})
			}

			if state == READING_BLOCK_DATA {
				return res, fmt.Errorf("Shard %d in progress", req.ShardId)
			} else if req.Begin < m.GetReliableDataStartTime() {
				return res, fmt.Errorf("missing too much data")
			}

			return res, nil

		}
	*/
}
