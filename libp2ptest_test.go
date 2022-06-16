package main

import "testing"

func TestGetmemuseinfo(t *testing.T) {
	memi, err := memuseinfo()
	if err != nil {
		t.Fatalf("memuseinfo error:%v", err)
	}
	t.Log("meminfo:", memi)
}

func TestGetcpuinfo(t *testing.T) {
	cpui, err := cpuinfo()
	if err != nil {
		t.Fatalf("cpuinfo error:%v", err)
	}
	t.Log("cpui:", cpui)
}
func TestGetDiskInfo(t *testing.T) {
	diski, err := getDiskInfo()
	if err != nil {
		t.Fatalf("getDiskInfo error:%v", err)
	}
	t.Log("getDiskInfo:", diski)
}

func TestGetreadDB(t *testing.T) {
	dbdata := readAndWriteDB(nil)

	t.Log("dbdata:", dbdata)
}
