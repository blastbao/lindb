// Licensed to LinDB under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. LinDB licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package config

import (
	"fmt"
	"time"

	"github.com/lindb/lindb/pkg/ltoml"
)

// HTTP represents a HTTP level configuration of broker.
type HTTP struct {
	Port         uint16         `toml:"port"`
	IdleTimeout  ltoml.Duration `toml:"idle-timeout"`
	WriteTimeout ltoml.Duration `toml:"write-timeout"`
	ReadTimeout  ltoml.Duration `toml:"read-timeout"`
}

func (h *HTTP) TOML() string {
	return fmt.Sprintf(`
	## Controls how HTTP Server are configured.
	##
	## which port broker's HTTP Server is listening on 
	port = %d
	## maximum duration the server should keep established connections alive.
	## Default: 5s
	idle-timeout = "%s"
	## maximum duration before timing out for server writes of the response
	## Default: 5s
	write-timeout = "%s"	
	## maximum duration for reading the entire request, including the body.
	## Default: 5s
	read-timeout = "%s"`,
		h.Port,
		h.IdleTimeout.Duration().String(),
		h.WriteTimeout.Duration().String(),
		h.ReadTimeout.Duration().String(),
	)
}

type Ingestion struct {
	IngestTimeout ltoml.Duration `toml:"ingest-timeout"`
}

func (i *Ingestion) TOML() string {
	return fmt.Sprintf(`
	## maximum duration before timeout for server ingesting metrics
	## Default: 5s
	ingest-timeout = "%s"`,
		i.IngestTimeout.Duration().String())
}

// User represents user model
type User struct {
	UserName string `toml:"username" json:"username" binding:"required"`
	Password string `toml:"password" json:"password" binding:"required"`
}

func (u *User) TOML() string {
	return fmt.Sprintf(`
	## admin user setting
	username = "%s"
	password = "%s"`,
		u.UserName,
		u.Password)
}

// ReplicationChannel represents config for data replication in broker.
type ReplicationChannel struct {
	Dir                string         `toml:"dir"`
	DataSizeLimit      int64          `toml:"data-size-limit"`
	RemoveTaskInterval ltoml.Duration `toml:"remove-task-interval"`
	ReportInterval     ltoml.Duration `toml:"report-interval"` // replicator state report interval
	CheckFlushInterval ltoml.Duration `toml:"check-flush-interval"`
	FlushInterval      ltoml.Duration `toml:"flush-interval"`
	BufferSize         int            `toml:"buffer-size"`
}

func (rc *ReplicationChannel) GetDataSizeLimit() int64 {
	if rc.DataSizeLimit <= 1 {
		return 1024 * 1024 // 1MB
	}
	if rc.DataSizeLimit >= 1024 {
		return 1024 * 1024 * 1024 // 1GB
	}
	return rc.DataSizeLimit * 1024 * 1024
}

func (rc *ReplicationChannel) BufferSizeInBytes() int {
	return rc.BufferSize
}

func (rc *ReplicationChannel) TOML() string {
	return fmt.Sprintf(`
	## WAL mmaped log directory
	dir = "%s"
	## data-size-limit is the maximum size in megabytes of the page file before a new
	## file is created. It defaults to 512 megabytes, available size is in [1MB, 1GB]
	data-size-limit = %d
	## interval for how often a new segment will be created
	remove-task-interval = "%s"
	## replicator state report interval
	report-interval = "%s"
	## interval for how often buffer will be checked if it's available to flush
	check-flush-interval = "%s"
	## interval for how often data will be flushed if data not exceeds the buffer-size
	flush-interval = "%s"
	## will flush if this size of data in kegabytes get buffered
	buffer-size = %d`,
		rc.Dir,
		rc.DataSizeLimit,
		rc.RemoveTaskInterval.String(),
		rc.ReportInterval.String(),
		rc.CheckFlushInterval.String(),
		rc.FlushInterval.String(),
		rc.BufferSize,
	)
}

// BrokerBase represents a broker configuration
type BrokerBase struct {
	HTTP      HTTP      `toml:"http"`
	Ingestion Ingestion `toml:"ingestion"`
	User      User      `toml:"user"`
	GRPC      GRPC      `toml:"grpc"`
}

func (bb *BrokerBase) TOML() string {
	return fmt.Sprintf(`[broker]
  [broker.http]%s

  [broker.ingestion]%s

  [broker.user]%s

  [broker.grpc]%s`,
		bb.HTTP.TOML(),
		bb.Ingestion.TOML(),
		bb.User.TOML(),
		bb.GRPC.TOML(),
	)
}

func NewDefaultBrokerBase() *BrokerBase {
	return &BrokerBase{
		HTTP: HTTP{
			Port:         9000,
			IdleTimeout:  ltoml.Duration(time.Minute * 2),
			ReadTimeout:  ltoml.Duration(time.Second * 5),
			WriteTimeout: ltoml.Duration(time.Second * 5),
		},
		Ingestion: Ingestion{
			IngestTimeout: ltoml.Duration(time.Second * 5),
		},
		GRPC: GRPC{
			Port:                 9001,
			TTL:                  ltoml.Duration(time.Second),
			MaxConcurrentStreams: 30,
			ConnectTimeout:       ltoml.Duration(time.Second * 3),
		},
		User: User{
			UserName: "admin",
			Password: "admin123",
		},
	}
}

// Broker represents a broker configuration with common settings
type Broker struct {
	Coordinator RepoState  `toml:"coordinator"`
	Query       Query      `toml:"query"`
	BrokerBase  BrokerBase `toml:"broker"`
	Monitor     Monitor    `toml:"monitor"`
	Logging     Logging    `toml:"logging"`
}

// NewDefaultBrokerTOML creates broker default toml config
func NewDefaultBrokerTOML() string {
	return fmt.Sprintf(`%s

%s

%s

%s

%s`,
		NewDefaultCoordinator().TOML(),
		NewDefaultQuery().TOML(),
		NewDefaultBrokerBase().TOML(),
		NewDefaultMonitor().TOML(),
		NewDefaultLogging().TOML(),
	)
}

func checkBrokerBaseCfg(brokerBaseCfg *BrokerBase) error {
	if err := checkGRPCCfg(&brokerBaseCfg.GRPC); err != nil {
		return err
	}
	defaultBrokerCfg := NewDefaultBrokerBase()
	// http check
	if brokerBaseCfg.HTTP.Port <= 0 {
		return fmt.Errorf("http port cannot be empty")
	}
	if brokerBaseCfg.HTTP.ReadTimeout <= 0 {
		brokerBaseCfg.HTTP.ReadTimeout = defaultBrokerCfg.HTTP.ReadTimeout
	}
	if brokerBaseCfg.HTTP.WriteTimeout <= 0 {
		brokerBaseCfg.HTTP.WriteTimeout = defaultBrokerCfg.HTTP.WriteTimeout
	}
	if brokerBaseCfg.HTTP.IdleTimeout <= 0 {
		brokerBaseCfg.HTTP.IdleTimeout = defaultBrokerCfg.HTTP.IdleTimeout
	}

	// ingestion
	if brokerBaseCfg.Ingestion.IngestTimeout <= 0 {
		brokerBaseCfg.Ingestion.IngestTimeout = defaultBrokerCfg.Ingestion.IngestTimeout
	}
	return nil
}
