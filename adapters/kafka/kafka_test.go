package kafka

import "testing"

func Test_route_address(t *testing.T) {
	brokers, topic, err := parseRouteAddress("broker1:9020,broker2:9020/hello")
	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	if len(brokers) != 2 {
		t.Fatal("expected two broker addrs")
	}
	if brokers[0] != "broker1:9020" {
		t.Errorf("broker1 addr should not be %s", brokers[0])
	}
	if brokers[1] != "broker2:9020" {
		t.Errorf("broker2 addr should not be %s", brokers[1])
	}
	if topic != "hello" {
		t.Errorf("topic should not be %s", topic)
	}
}

func Test_route_address_is_missing_a_topic(t *testing.T) {
	_, _, err := parseRouteAddress("broker1:9020,broker2:9020")
	if err == nil {
		t.Errorf("expected an error for a missing topic")
	}
}
