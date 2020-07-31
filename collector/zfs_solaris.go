// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build solaris,!illumos
// +build !nozfs

package collector

import (
	"errors"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/siebenmann/go-kstat"
)

type zfsCollector struct {
	arcstatsC                    *prometheus.Desc
	arcstatsCMax                 *prometheus.Desc
	arcstatsCMin                 *prometheus.Desc
	arcstatsDataSize             *prometheus.Desc
	arcstatsDemandDataHits       *prometheus.Desc
	arcstatsDemandDataMisses     *prometheus.Desc
	arcstatsDemandMetadataHits   *prometheus.Desc
	arcstatsDemandMetadataMisses *prometheus.Desc
	arcstatsHits                 *prometheus.Desc
	arcstatsMisses               *prometheus.Desc
	arcstatsMFUGhostHits         *prometheus.Desc
	arcstatsMRUGhostHits         *prometheus.Desc
	arcstatsOtherSize            *prometheus.Desc
	arcstatsP                    *prometheus.Desc
	arcstatsSize                 *prometheus.Desc
	logger                       log.Logger
}

const (
	zfsCollectorSubsystem = "zfs"
)

func init() {
	registerCollector("zfs", defaultEnabled, NewZfsCollector)
}

func NewZfsCollector(logger log.Logger) (Collector, error) {
	return &zfsCollector{
		arcstatsC: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zfsCollectorSubsystem, "arcstats_c_bytes"),
			"ZFS ARC target size", nil, nil,
		),
		arcstatsCMax: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zfsCollectorSubsystem, "arcstats_c_max_bytes"),
			"ZFS ARC maximum size", nil, nil,
		),
		arcstatsCMin: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zfsCollectorSubsystem, "arcstats_c_min_bytes"),
			"ZFS ARC minimum size", nil, nil,
		),
		arcstatsDataSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zfsCollectorSubsystem, "arcstats_data_bytes"),
			"ZFS ARC data size", nil, nil,
		),
		arcstatsDemandDataHits: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zfsCollectorSubsystem, "arcstats_demand_data_hits_total"),
			"ZFS ARC demand data hits", nil, nil,
		),
		arcstatsDemandDataMisses: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zfsCollectorSubsystem, "arcstats_demand_data_misses_total"),
			"ZFS ARC demand data misses", nil, nil,
		),
		arcstatsDemandMetadataHits: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zfsCollectorSubsystem, "arcstats_demand_metadata_hits_total"),
			"ZFS ARC demand metadata hits", nil, nil,
		),
		arcstatsDemandMetadataMisses: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zfsCollectorSubsystem, "arcstats_demand_metadata_misses_total"),
			"ZFS ARC demand metadata misses", nil, nil,
		),
		arcstatsHits: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zfsCollectorSubsystem, "arcstats_hits_total"),
			"ZFS ARC hits", nil, nil,
		),
		arcstatsMisses: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zfsCollectorSubsystem, "arcstats_misses_total"),
			"ZFS ARC misses", nil, nil,
		),
		arcstatsMFUGhostHits: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zfsCollectorSubsystem, "arcstats_mfu_ghost_hits_total"),
			"ZFS ARC MFU ghost hits", nil, nil,
		),
		arcstatsMRUGhostHits: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zfsCollectorSubsystem, "arcstats_mru_ghost_hits_total"),
			"ZFS ARC MRU ghost hits", nil, nil,
		),
		arcstatsOtherSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zfsCollectorSubsystem, "arcstats_other_bytes"),
			"ZFS ARC other size", nil, nil,
		),
		arcstatsP: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zfsCollectorSubsystem, "arcstats_p_bytes"),
			"ZFS ARC MRU target size", nil, nil,
		),
		arcstatsSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zfsCollectorSubsystem, "arcstats_size_bytes"),
			"ZFS ARC size", nil, nil,
		),
		logger: logger,
	}, nil
}

func (c *zfsCollector) updateZfsArcStats(ch chan<- prometheus.Metric) error {
	var metricType prometheus.ValueType

	tok, err := kstat.Open()
	if err != nil {
		return err
	}

	defer tok.Close()

	ksZFSInfo, err := tok.Lookup("zfs", 0, "arcstats")
	if err != nil {
		return err
	}

	for k, v := range map[string]*prometheus.Desc{
		"c":                      c.arcstatsC,
		"c_max":                  c.arcstatsCMax,
		"c_min":                  c.arcstatsCMin,
		"data_size":              c.arcstatsDataSize,
		"demand_data_hits":       c.arcstatsDemandDataHits,
		"demand_data_misses":     c.arcstatsDemandDataMisses,
		"demand_metadata_hits":   c.arcstatsDemandMetadataHits,
		"demand_metadata_misses": c.arcstatsDemandMetadataMisses,
		"hits":                   c.arcstatsHits,
		"misses":                 c.arcstatsMisses,
		"mfu_ghost_hits":         c.arcstatsMFUGhostHits,
		"mru_ghost_hits":         c.arcstatsMRUGhostHits,
		"other_size":             c.arcstatsOtherSize,
		"p":                      c.arcstatsP,
		"size":                   c.arcstatsSize,
	} {
		ksZFSInfoValue, err := ksZFSInfo.GetNamed(k)
		if err != nil {
			return errors.New("ksZFSInfo.GetNamed(" + k + "): " + err.Error())
		}

		if strings.HasSuffix(k, "_hits") || strings.HasSuffix(k, "_misses") {
			metricType = prometheus.CounterValue
		} else {
			metricType = prometheus.GaugeValue
		}

		ch <- prometheus.MustNewConstMetric(
			v,
			metricType,
			float64(ksZFSInfoValue.UintVal),
		)
	}

	return nil
}

func (c *zfsCollector) Update(ch chan<- prometheus.Metric) error {
	if err := c.updateZfsArcStats(ch); err != nil {
		return errors.New("updateZfsArcStats: " + err.Error())
	}
	return nil
}
