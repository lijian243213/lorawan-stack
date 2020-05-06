// Copyright © 2019 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mqtt_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogo/protobuf/proto"
	"github.com/smartystreets/assertions"
	"go.thethings.network/lorawan-stack/v3/pkg/cluster"
	"go.thethings.network/lorawan-stack/v3/pkg/component"
	componenttest "go.thethings.network/lorawan-stack/v3/pkg/component/test"
	"go.thethings.network/lorawan-stack/v3/pkg/config"
	"go.thethings.network/lorawan-stack/v3/pkg/errors"
	"go.thethings.network/lorawan-stack/v3/pkg/frequencyplans"
	"go.thethings.network/lorawan-stack/v3/pkg/gatewayserver/io"
	"go.thethings.network/lorawan-stack/v3/pkg/gatewayserver/io/mock"
	. "go.thethings.network/lorawan-stack/v3/pkg/gatewayserver/io/mqtt"
	"go.thethings.network/lorawan-stack/v3/pkg/log"
	"go.thethings.network/lorawan-stack/v3/pkg/ttnpb"
	"go.thethings.network/lorawan-stack/v3/pkg/unique"
	"go.thethings.network/lorawan-stack/v3/pkg/util/test"
	"go.thethings.network/lorawan-stack/v3/pkg/util/test/assertions/should"
)

var (
	registeredGatewayID  = ttnpb.GatewayIdentifiers{GatewayID: "test-gateway"}
	registeredGatewayUID = unique.ID(test.Context(), registeredGatewayID)
	registeredGatewayKey = "test-key"

	timeout = 20 * test.Delay
)

func TestAuthentication(t *testing.T) {
	a := assertions.New(t)

	ctx := log.NewContext(test.Context(), test.GetLogger(t))
	ctx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()

	is, isAddr := mock.NewIS(ctx)
	is.Add(ctx, registeredGatewayID, registeredGatewayKey)

	c := componenttest.NewComponent(t, &component.Config{
		ServiceBase: config.ServiceBase{
			GRPC: config.GRPC{
				Listen:                      ":0",
				AllowInsecureForCredentials: true,
			},
			Cluster: cluster.Config{
				IdentityServer: isAddr,
			},
		},
	})
	c.FrequencyPlans = frequencyplans.NewStore(test.FrequencyPlansFetcher)
	componenttest.StartComponent(t, c)
	defer c.Close()
	mustHavePeer(ctx, c, ttnpb.ClusterRole_ENTITY_REGISTRY)

	gs := mock.NewServer(c)
	lis, err := net.Listen("tcp", ":0")
	if !a.So(err, should.BeNil) {
		t.FailNow()
	}
	go Serve(ctx, gs, lis, NewProtobuf(ctx), "tcp")

	for _, tc := range []struct {
		UID string
		Key string
		OK  bool
	}{
		{
			UID: registeredGatewayUID,
			Key: registeredGatewayKey,
			OK:  true,
		},
		{
			UID: registeredGatewayUID,
			Key: "invalid-key",
			OK:  false,
		},
		{
			UID: "invalid-gateway",
			Key: "invalid-key",
			OK:  false,
		},
	} {
		t.Run(fmt.Sprintf("%v:%v", tc.UID, tc.Key), func(t *testing.T) {
			a := assertions.New(t)

			clientOpts := mqtt.NewClientOptions()
			clientOpts.AddBroker(fmt.Sprintf("tcp://%v", lis.Addr()))
			clientOpts.SetUsername(tc.UID)
			clientOpts.SetPassword(tc.Key)
			client := mqtt.NewClient(clientOpts)
			token := client.Connect()
			if tc.OK {
				if !token.WaitTimeout(timeout) {
					t.Fatal("Connection timeout")
				}
				if !a.So(token.Error(), should.BeNil) {
					t.FailNow()
				}
			} else if token.Wait() && !a.So(token.Error(), should.NotBeNil) {
				t.FailNow()
			}
			client.Disconnect(uint(timeout / time.Millisecond))
		})
	}
}

func TestTraffic(t *testing.T) {
	a := assertions.New(t)

	ctx := log.NewContext(test.Context(), test.GetLogger(t))
	ctx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()

	is, isAddr := mock.NewIS(ctx)
	is.Add(ctx, registeredGatewayID, registeredGatewayKey)

	c := componenttest.NewComponent(t, &component.Config{
		ServiceBase: config.ServiceBase{
			GRPC: config.GRPC{
				Listen:                      ":0",
				AllowInsecureForCredentials: true,
			},
			Cluster: cluster.Config{
				IdentityServer: isAddr,
			},
		},
	})
	c.FrequencyPlans = frequencyplans.NewStore(test.FrequencyPlansFetcher)
	componenttest.StartComponent(t, c)
	defer c.Close()
	mustHavePeer(ctx, c, ttnpb.ClusterRole_ENTITY_REGISTRY)

	gs := mock.NewServer(c)
	lis, err := net.Listen("tcp", ":0")
	if !a.So(err, should.BeNil) {
		t.FailNow()
	}
	go Serve(ctx, gs, lis, NewProtobuf(ctx), "tcp")

	clientOpts := mqtt.NewClientOptions()
	clientOpts.AddBroker(fmt.Sprintf("tcp://%v", lis.Addr()))
	clientOpts.SetUsername(registeredGatewayUID)
	clientOpts.SetPassword(registeredGatewayKey)
	client := mqtt.NewClient(clientOpts)
	token := client.Connect()
	if !token.WaitTimeout(timeout) {
		t.Fatal("Connection timeout")
	}
	if !a.So(token.Error(), should.BeNil) {
		t.FailNow()
	}

	var conn *io.Connection
	select {
	case conn = <-gs.Connections():
	case <-time.After(timeout):
		t.Fatal("Connection timeout")
	}
	defer client.Disconnect(100)

	t.Run("Upstream", func(t *testing.T) {
		for _, tc := range []struct {
			Topic   string
			Message proto.Marshaler
			OK      bool
		}{
			{
				Topic: fmt.Sprintf("v3/%v/up", registeredGatewayUID),
				Message: &ttnpb.UplinkMessage{
					RawPayload: []byte{0x01},
				},
				OK: true,
			},
			{
				Topic: fmt.Sprintf("v3/%v/up", "invalid-gateway"),
				Message: &ttnpb.UplinkMessage{
					RawPayload: []byte{0x03},
				},
				OK: false, // invalid gateway ID
			},
			{
				Topic: fmt.Sprintf("v3/%v/down", registeredGatewayUID),
				Message: &ttnpb.DownlinkMessage{
					RawPayload: []byte{0x04},
				},
				OK: false, // publish to downlink not permitted
			},
			{
				Topic: fmt.Sprintf("v3/%v/down", "invalid-gateway"),
				Message: &ttnpb.DownlinkMessage{
					RawPayload: []byte{0x05},
				},
				OK: false, // invalid gateway ID + publish to downlink not permitted
			},
			{
				Topic: fmt.Sprintf("v3/%v/status", registeredGatewayUID),
				Message: &ttnpb.GatewayStatus{
					IP: []string{"1.1.1.1"},
				},
				OK: true,
			},
			{
				Topic: fmt.Sprintf("v3/%v/down/ack", registeredGatewayUID),
				Message: &ttnpb.TxAcknowledgment{
					Result: ttnpb.TxAcknowledgment_SUCCESS,
				},
				OK: true,
			},
			{
				Topic: fmt.Sprintf("v3/%v/status", "invalid-gateway"),
				Message: &ttnpb.GatewayStatus{
					IP: []string{"2.2.2.2"},
				},
				OK: false, // invalid gateway ID
			},
			{
				Topic: "invalid/format",
				Message: &ttnpb.GatewayStatus{
					IP: []string{"3.3.3.3"},
				},
				OK: false, // invalid topic format
			},
		} {
			tcok := t.Run(tc.Topic, func(t *testing.T) {
				a := assertions.New(t)
				buf, err := tc.Message.Marshal()
				a.So(err, should.BeNil)
				token := client.Publish(tc.Topic, 1, false, buf)
				if !token.WaitTimeout(timeout) {
					t.Fatal("Publish timeout")
				}
				if !a.So(token.Error(), should.BeNil) {
					t.FailNow()
				}
				select {
				case up := <-conn.Up():
					if tc.OK {
						a.So(time.Since(up.ReceivedAt), should.BeLessThan, timeout)
						up.ReceivedAt = time.Time{}
						a.So(up.UplinkMessage, should.Resemble, tc.Message)
					} else {
						t.Fatalf("Did not expect uplink message, but have %v", up)
					}
				case status := <-conn.Status():
					if tc.OK {
						a.So(status, should.Resemble, tc.Message)
					} else {
						t.Fatalf("Did not expect status message, but have %v", status)
					}
				case ack := <-conn.TxAck():
					if tc.OK {
						a.So(ack, should.Resemble, tc.Message)
					} else {
						t.Fatalf("Did not expect ack message, but have %v", ack)
					}
				case <-time.After(timeout):
					if tc.OK {
						t.Fatal("Receive expected upstream timeout")
					}
				}
			})
			if !tcok {
				t.FailNow()
			}
		}
	})

	t.Run("Downstream", func(t *testing.T) {
		for _, tc := range []struct {
			Topic          string
			Path           *ttnpb.DownlinkPath
			Message        *ttnpb.DownlinkMessage
			ErrorAssertion func(error) bool
		}{
			{
				Topic: fmt.Sprintf("v3/%v/down", registeredGatewayUID),
				Path: &ttnpb.DownlinkPath{
					Path: &ttnpb.DownlinkPath_UplinkToken{
						UplinkToken: io.MustUplinkToken(ttnpb.GatewayAntennaIdentifiers{GatewayIdentifiers: registeredGatewayID}, 100),
					},
				},
				Message: &ttnpb.DownlinkMessage{
					RawPayload: []byte{0x01},
					Settings: &ttnpb.DownlinkMessage_Request{
						Request: &ttnpb.TxRequest{
							Class:            ttnpb.CLASS_A,
							Priority:         ttnpb.TxSchedulePriority_NORMAL,
							Rx1Delay:         ttnpb.RX_DELAY_1,
							Rx1DataRateIndex: 5,
							Rx1Frequency:     868100000,
							Rx2DataRateIndex: 0,
							Rx2Frequency:     869525000,
							FrequencyPlanID:  test.EUFrequencyPlanID,
						},
					},
				},
			},
			{
				Topic: fmt.Sprintf("v3/%v/down", registeredGatewayUID),
				Path: &ttnpb.DownlinkPath{
					Path: &ttnpb.DownlinkPath_UplinkToken{
						UplinkToken: io.MustUplinkToken(ttnpb.GatewayAntennaIdentifiers{GatewayIdentifiers: registeredGatewayID}, 100),
					},
				},
				Message: &ttnpb.DownlinkMessage{
					RawPayload: []byte{0x01},
					Settings: &ttnpb.DownlinkMessage_Scheduled{
						Scheduled: &ttnpb.TxSettings{
							DataRate: ttnpb.DataRate{
								Modulation: &ttnpb.DataRate_LoRa{
									LoRa: &ttnpb.LoRaDataRate{
										Bandwidth:       125000,
										SpreadingFactor: 7,
									},
								},
							},
							CodingRate: "4/5",
							Frequency:  869525000,
						},
					},
				},
				ErrorAssertion: errors.IsInvalidArgument, // Does not support scheduled downlink; the Gateway Server or gateway will take care of that.
			},
			{
				Topic: fmt.Sprintf("v3/%v/down", registeredGatewayUID),
				Path: &ttnpb.DownlinkPath{
					Path: &ttnpb.DownlinkPath_UplinkToken{
						UplinkToken: io.MustUplinkToken(ttnpb.GatewayAntennaIdentifiers{GatewayIdentifiers: registeredGatewayID}, 100),
					},
				},
				Message: &ttnpb.DownlinkMessage{
					RawPayload: []byte{0x02},
				},
				ErrorAssertion: errors.IsInvalidArgument, // Tx request missing.
			},
		} {
			t.Run(tc.Topic, func(t *testing.T) {
				a := assertions.New(t)

				downCh := make(chan *ttnpb.DownlinkMessage)
				handler := func(_ mqtt.Client, msg mqtt.Message) {
					down := &ttnpb.GatewayDown{}
					err := down.Unmarshal(msg.Payload())
					a.So(err, should.BeNil)
					downCh <- down.DownlinkMessage
				}
				token := client.Subscribe(tc.Topic, 1, handler)
				if !token.WaitTimeout(timeout) {
					t.Fatal("Subscribe timeout")
				}
				if !a.So(token.Error(), should.BeNil) {
					t.FailNow()
				}

				_, err := conn.ScheduleDown(tc.Path, tc.Message)
				if err != nil && (tc.ErrorAssertion == nil || !a.So(tc.ErrorAssertion(err), should.BeTrue)) {
					t.Fatalf("Unexpected error: %v", err)
				}
				select {
				case down := <-downCh:
					if tc.ErrorAssertion == nil {
						a.So(down, should.Resemble, tc.Message)
					} else {
						t.Fatalf("Unexpected message: %v", down)
					}
				case <-time.After(timeout):
					if tc.ErrorAssertion == nil {
						t.Fatal("Receive expected downlink timeout")
					}
				}
			})
		}
	})
}
