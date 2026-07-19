// Copyright 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dyndns

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/libdns/libdns"
)

func TestUpdate(t *testing.T) {
	const zone = "zekjur.net"
	update := libdns.Address{
		Name: "dyndns.zekjur.net",
		IP:   netip.MustParseAddr("127.0.0.4"),
		TTL:  5 * time.Minute,
	}
	// Return zone-relative names (e.g. dyndns, instead of dyndns.zekjur.net),
	// like the real cloudflare libdns provider.
	unrelated := libdns.Address{
		Name: "unrelated",
		TTL:  5 * time.Minute,
		IP:   netip.MustParseAddr("127.0.0.42"),
	}
	for _, tt := range []struct {
		name     string
		existing []libdns.Record
		want     []libdns.RR // nil means SetRecords must not be called
	}{
		{
			name: "update existing record",
			existing: []libdns.Record{
				libdns.Address{
					Name: "dyndns",
					TTL:  5 * time.Minute,
					IP:   netip.MustParseAddr("127.0.0.3"),
				},
				unrelated,
			},
			want: []libdns.RR{
				{
					Name: "dyndns",
					Type: "A",
					Data: "127.0.0.4",
					TTL:  5 * time.Minute,
				},
			},
		},

		{
			name: "record up to date",
			existing: []libdns.Record{
				libdns.Address{
					Name: "dyndns",
					TTL:  5 * time.Minute,
					IP:   netip.MustParseAddr("127.0.0.4"),
				},
			},
			want: nil,
		},

		{
			name:     "create missing record",
			existing: []libdns.Record{unrelated},
			want: []libdns.RR{
				{
					Name: "dyndns",
					Type: "A",
					Data: "127.0.0.4",
					TTL:  5 * time.Minute,
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var calls [][]libdns.Record
			provider := &testProvider{
				getRecords: func(ctx context.Context, zone string) ([]libdns.Record, error) {
					return tt.existing, nil
				},
				setRecords: func(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
					calls = append(calls, recs)
					return nil, nil
				},
			}
			if err := Update(context.Background(), zone, update, provider); err != nil {
				t.Fatalf("Update = %v", err)
			}
			if tt.want == nil {
				if len(calls) > 0 {
					t.Fatalf("SetRecords unexpectedly called with %+v", calls)
				}
				return
			}
			if got, want := len(calls), 1; got != want {
				t.Fatalf("SetRecords called %d times, want %d", got, want)
			}
			var got []libdns.RR
			for _, rec := range calls[0] {
				got = append(got, rec.RR())
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("SetRecords: unexpected records: diff (-want +got):\n%s", diff)
			}
		})
	}
}

var (
	_ RecordGetterSetter = &testProvider{}
)

type testProvider struct {
	getRecords func(ctx context.Context, zone string) ([]libdns.Record, error)
	setRecords func(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error)
}

func (p *testProvider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	return p.getRecords(ctx, zone)
}

func (p *testProvider) SetRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	return p.setRecords(ctx, zone, recs)
}
