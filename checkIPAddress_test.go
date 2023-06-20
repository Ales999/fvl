package main

import "testing"

func Test_checkIPAddress(t *testing.T) {
	type args struct {
		ip string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.

		{name: "Check Valid IP Test1", args: args{"192.168.1.1"}, want: true},
		{name: "Check Valid IP Test2", args: args{"192.168.1.1"}, want: true},
		{name: "Check InValid IP Test1", args: args{"1270.1.0.1"}, want: false},
		{name: "Check InValid IP Test2", args: args{"127*0*0*1"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkIPAddress(tt.args.ip); got != tt.want {
				t.Errorf("checkIPAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}
