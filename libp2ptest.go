package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	peerstore "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	_ "github.com/mattn/go-sqlite3"
	"github.com/multiformats/go-multiaddr"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

var ProtocolID protocol.ID = "test/1.0.0"

type Stat struct {
	Direction network.Direction
	Opend     time.Time
	Transient bool
}

type DbData struct {
	UserId int
	Uname  string
}

type NodeInfo struct {
	h         host.Host
	Stats     Stat                   `json:"stats"`
	Nodeslist []string               `json:"nodeslist"`
	Meminfo   *mem.VirtualMemoryStat `json:"meminfo"`
	Cpuinfo   []cpu.TimesStat        `json:"cpuinfo"`
	Diskinfo  []*disk.UsageStat      `json:"diskinfo"`
	Dbdata    []DbData               `json:"Dbdata"`
}

func main() {
	node, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.Ping(false),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("node.ID:", node.ID().String())
	peerInfo := peerstore.AddrInfo{
		ID:    node.ID(),
		Addrs: node.Addrs(),
	}
	addrs, err := peerstore.AddrInfoToP2pAddrs(&peerInfo)
	if err != nil {
		panic(err)
	}
	fmt.Println("libp2p node address:", addrs[0])

	nodeinfo := &NodeInfo{}
	nodeinfo.h = node
	node.SetStreamHandler(ProtocolID, nodeinfo.handleStream)

	if len(os.Args) > 1 {
		addr, err := multiaddr.NewMultiaddr(os.Args[1])
		if err != nil {
			panic(err)
		}
		peer, err := peerstore.AddrInfoFromP2pAddr(addr)
		if err != nil {
			panic(err)
		}
		if err := node.Connect(context.Background(), *peer); err != nil {
			fmt.Println("Connection failed:", err)
			panic(err)
		}
		fmt.Println("peer.ID:", peer.ID)
		s, err := node.NewStream(context.Background(), peer.ID, ProtocolID)
		if err != nil {
			panic(err)
		}

		nodeinfo.handleStream(s)
	}
	select {}
}

func (nd *NodeInfo) handleStream(s network.Stream) {
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	go readData(rw)
	wrch := make(chan string)
	go getAllData(s, nd, wrch)
	go writeData(rw, wrch)

}

func readData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from buffer")
			return
		}

		if str == "" {
			continue
		}
		if str != "\n" {
			//fmt.Printf("readData: %v\n", str)
			var nd NodeInfo

			err := json.Unmarshal([]byte(str), &nd)
			if err != nil {
				fmt.Println("Error Unmarshal")
				return
			}
			fmt.Printf("readData: %v\n", nd)
		}
		time.Sleep(time.Second)
	}
}

func writeData(rw *bufio.ReadWriter, wrch chan string) {
	defer close(wrch)
	for {
		select {
		case data, ok := <-wrch:
			if !ok {
				return
			}
			sdata := data + "\n"
			_, err := rw.WriteString(sdata)
			if err != nil {
				fmt.Println("Error writing to buffer")
				return
			}
			fmt.Println("sendData:", sdata)
			err = rw.Flush()
			if err != nil {
				fmt.Println("Error flushing buffer")
				return
			}
		}
	}

}

func getAllData(s network.Stream, nd *NodeInfo, wrch chan string) {
	st := getState(s)
	nd.Stats.Direction = st.Direction
	nd.Stats.Opend = st.Opened
	nd.Stats.Transient = st.Transient

	nd.Nodeslist = getNodesList(nd.h)

	memifo, err := memuseinfo()
	if err != nil {
		fmt.Println("memuseinfo error:", err)
		s.Reset()
		return
	}
	nd.Meminfo = memifo

	cpuinfo, err := cpuinfo()
	if err != nil {
		fmt.Println("cpuinfo error:", err)
		s.Reset()
		return
	}
	nd.Cpuinfo = cpuinfo

	diskinfo, err := getDiskInfo()
	if err != nil {
		fmt.Println("getDiskInfo error:", err)
		s.Reset()
		return
	}
	nd.Diskinfo = diskinfo

	nd.Dbdata = readAndWriteDB(nd)

	sendData, err := json.Marshal(nd)
	if err != nil {
		fmt.Println("Marshal:", err)
		s.Reset()
		return
	}
	wrch <- string(sendData)
	return
}

func getState(s network.Stream) network.Stats {
	return s.Stat()
}

func getNodesList(h host.Host) []string {
	var addrs []string
	for _, a := range h.Addrs() {
		addrs = append(addrs, a.String())
	}
	return addrs
}

func memuseinfo() (*mem.VirtualMemoryStat, error) {
	return mem.VirtualMemory()
}

func cpuinfo() ([]cpu.TimesStat, error) {
	return cpu.Times(false)
}

func getDiskInfo() ([]*disk.UsageStat, error) {
	parts, err := disk.Partitions(true)
	if err != nil {
		fmt.Printf("get Partitions failed, err:%v", err)
		return nil, err
	}
	var diskpartsinfo []*disk.UsageStat
	for _, part := range parts {
		diskInfo, err := disk.Usage(part.Mountpoint)
		if err != nil {
			return nil, err
		}
		diskpartsinfo = append(diskpartsinfo, diskInfo)
	}
	return diskpartsinfo, nil
}

func readAndWriteDB(ndinfo *NodeInfo) []DbData {
	os.Remove("./foo.db")
	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	//jWT
	//sql注入
	//ROM
	//预编译

	sql := `create table nodeinfo (id integer, state text,nodelist text,mem text,cpu text,disk text);`
	db.Exec(sql)
	sql = `insert into users(id,state,nodelist,mem,cpu,disk) values(...);`
	db.Exec(sql)
	sql = `insert into users(userId,uname) values(2,'Eirc');`
	db.Exec(sql)
	rows, err := db.Query("select * from users")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var dbdata []DbData = make([]DbData, 0)
	for rows.Next() {
		var u DbData
		rows.Scan(&u.UserId, &u.Uname)
		dbdata = append(dbdata, u)
	}
	fmt.Println(dbdata)
	return dbdata
}
