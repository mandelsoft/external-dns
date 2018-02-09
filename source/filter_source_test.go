/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package source

import (
	"net"
	"testing"

	"github.com/kubernetes-incubator/external-dns/endpoint"
	"github.com/kubernetes-incubator/external-dns/internal/testutils"
	"github.com/kubernetes-incubator/external-dns/provider"
)

// Validates that filterSource is a Source
var _ Source = &filterSource{}

func TestFilter(t *testing.T) {
	t.Run("Filter Cidr Endpoints", testFilterCidrEndpoints)
	t.Run("Filter DNS Names", testFilterDNSNames)
}

// testFilterCidrEndpoints tests that filtered IPs from the wrapped source are removed.
func testFilterCidrEndpoints(t *testing.T) {
	for _, tc := range []struct {
		title     string
		endpoints []*endpoint.Endpoint
		expected  []*endpoint.Endpoint
	}{
		{
			"one endpoint returns one endpoint",
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
		},
		{
			"two different endpoints return two endpoints",
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
				{DNSName: "bar.example.org", Target: "4.5.6.7", RecordType: endpoint.RecordTypeA},
			},
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
				{DNSName: "bar.example.org", Target: "4.5.6.7", RecordType: endpoint.RecordTypeA},
			},
		},
		{
			"non A-records ignores",
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
				{DNSName: "foo.example.org", Target: "192.168.100.10", RecordType: endpoint.RecordTypeCNAME},
			},
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
				{DNSName: "foo.example.org", Target: "192.168.100.10", RecordType: endpoint.RecordTypeCNAME},
			},
		},
		{
			"two endpoints with same dnsname and ignored target return one endpoint",
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
				{DNSName: "foo.example.org", Target: "192.168.100.10", RecordType: endpoint.RecordTypeA},
			},
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
		},
		{
			"two endpoints with different dnsname and ignored target return one endpoint",
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
				{DNSName: "bar.example.org", Target: "192.168.100.10", RecordType: endpoint.RecordTypeA},
			},
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			mockSource := new(testutils.MockSource)
			mockSource.On("Endpoints").Return(tc.endpoints, nil)

			// Create our object under test and get the endpoints.
			_, cidr, _ := net.ParseCIDR("192.168.100.0/24")
			source := NewFilterSource(provider.NewDomainFilter([]string{}), []*net.IPNet{cidr}, nil, mockSource)

			endpoints, err := source.Endpoints()
			if err != nil {
				t.Fatal(err)
			}

			// Validate returned endpoints against desired endpoints.
			validateEndpoints(t, endpoints, tc.expected)

			// Validate that the mock source was called.
			mockSource.AssertExpectations(t)
		})
	}
}

// testFilterDNS tests that filtered DNS names are removed.
func testFilterDNSNames(t *testing.T) {
	for _, tc := range []struct {
		title     string
		endpoints []*endpoint.Endpoint
		expected  []*endpoint.Endpoint
	}{
		{
			"one endpoint returns one endpoint",
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
		},
		{
			"two different endpoints return two endpoints",
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
				{DNSName: "bar.example.org", Target: "4.5.6.7", RecordType: endpoint.RecordTypeA},
			},
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
				{DNSName: "bar.example.org", Target: "4.5.6.7", RecordType: endpoint.RecordTypeA},
			},
		},
		{
			"endpoints with ignored dnsname are removed",
			[]*endpoint.Endpoint{
				{DNSName: "foo.bar.example.com", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
		},
		{
			"endpoints with dnsname ignored by wildcard are removed",
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.com", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
			[]*endpoint.Endpoint{
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
		},
		{
			"endpoints with dnsname with more levels than wildcard are not removed",
			[]*endpoint.Endpoint{
				{DNSName: "bar.foo.example.com", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
			[]*endpoint.Endpoint{
				{DNSName: "bar.foo.example.com", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
		},
		{
			"endpoints with matching wildcard are not removed",
			[]*endpoint.Endpoint{
				{DNSName: "*.example.com", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
			[]*endpoint.Endpoint{
				{DNSName: "*.example.com", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
				{DNSName: "foo.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			mockSource := new(testutils.MockSource)
			mockSource.On("Endpoints").Return(tc.endpoints, nil)

			// Create our object under test and get the endpoints.
			source := NewFilterSource(provider.NewDomainFilter([]string{}), nil, []string{"*.example.com", "foo.bar.example.com"}, mockSource)

			endpoints, err := source.Endpoints()
			if err != nil {
				t.Fatal(err)
			}

			// Validate returned endpoints against desired endpoints.
			validateEndpoints(t, endpoints, tc.expected)

			// Validate that the mock source was called.
			mockSource.AssertExpectations(t)
		})
	}
}

// testFilterBaseDomain tests that filtered DNS names are removed.
func testFilterBaseDomain(t *testing.T) {
	for _, tc := range []struct {
		title     string
		endpoints []*endpoint.Endpoint
		expected  []*endpoint.Endpoint
	}{
		{
			"matching endpoint returns one endpoint",
			[]*endpoint.Endpoint{
				{DNSName: "foo.cluster.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
			[]*endpoint.Endpoint{
				{DNSName: "foo.cluster.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
		},
		{
			"endpoints with ignored non matching dnsname are removed",
			[]*endpoint.Endpoint{
				{DNSName: "foo.bar.example.com", Target: "1.2.3.6", RecordType: endpoint.RecordTypeA},
				{DNSName: "foo.example.org", Target: "1.2.3.5", RecordType: endpoint.RecordTypeA},
				{DNSName: "foo.cluster.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
			[]*endpoint.Endpoint{
				{DNSName: "foo.cluster.example.org", Target: "1.2.3.4", RecordType: endpoint.RecordTypeA},
			},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			mockSource := new(testutils.MockSource)
			mockSource.On("Endpoints").Return(tc.endpoints, nil)

			// Create our object under test and get the endpoints.
			source := NewFilterSource(provider.NewDomainFilter([]string{"cluster.example.org"}), nil, []string{"*.example.com", "foo.bar.example.com"}, mockSource)

			endpoints, err := source.Endpoints()
			if err != nil {
				t.Fatal(err)
			}

			// Validate returned endpoints against desired endpoints.
			validateEndpoints(t, endpoints, tc.expected)

			// Validate that the mock source was called.
			mockSource.AssertExpectations(t)
		})
	}
}
