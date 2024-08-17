package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"time"
)

var (
	timeout      int64
	size         int
	count        int
	typ          uint8 = 8
	code         uint8 = 0
	sendCount    int
	successCount int
	failCount    int
	minTs        int64 = math.MaxInt64
	maxTs        int64
	totalTs      int64
)

type ICMP struct { //定義ICMP封包結構
	Type     uint8
	Code     uint8
	Checksum uint16
	ID       uint16
	Seq      uint16
}

func main() {
	getCommandArgs()                                                                        //呼叫 getCommandArgs() 函式
	fmt.Println("timeout:", timeout, "size:", size, "count:", count)                        //輸出參數
	desIp := os.Args[len(os.Args)-1]                                                        //取得最後一個參數為目的IP 這裡google.com  會直接存並沒有轉ip
	conn, err := net.DialTimeout("ip:icmp", desIp, time.Duration(timeout)*time.Millisecond) //建立icmp連線   這裡google.com 才會被net 自帶的轉換DNS轉換
	if err != nil {
		log.Fatal(err)
		return
	}
	defer conn.Close()
	fmt.Printf("正在Ping %s [%s] 具有%d byte的數據\n", desIp, conn.RemoteAddr(), size)

	for i := 0; i < count; i++ {
		sendCount++
		t1 := time.Now()
		icmp := &ICMP{
			Type:     typ,  // 8通常是echo request
			Code:     code, // 0通常是無特定代碼
			Checksum: 0,
			ID:       1,
			Seq:      1,
		}

		data := make([]byte, size)
		var buffer bytes.Buffer                                                     //定義緩衝區
		binary.Write(&buffer, binary.BigEndian, icmp)                               //將ICMP封包結構寫入緩衝區
		buffer.Write(data)                                                          //將資料寫入緩衝區 32個byte
		data = buffer.Bytes()                                                       //將緩衝區內容轉成byte陣列 且更新 data資料
		checkSum := checksum(data)                                                  //計算checksum
		data[2] = byte(checkSum >> 8)                                               //更新checksum 2主要是因為type 1byte code 1byte checksum 2byte 這邊是高位
		data[3] = byte(checkSum)                                                    //更新checksum 3主要是因為type 1byte code 1byte checksum 2byte 這邊是低位
		conn.SetDeadline(time.Now().Add(time.Duration(timeout) * time.Millisecond)) //設定超時時間
		n, err := conn.Write(data)                                                  //寫入icmp封包 這裡 n 代表實際發送的byte數

		if err != nil {
			failCount++
			log.Println("請求超時", err)
			continue
		}
		buf := make([]byte, 65535)
		n, err = conn.Read(buf) // n是回應的byte數 65535是最大值 因為icmp最大封包大小是65535 buf 就是個工具人負責接收的緩存
		if err != nil {
			failCount++
			log.Println("回應超時", err)
			continue
		}
		successCount++
		ts := time.Since(t1).Milliseconds() //從當前時間到t1這多少時間

		if minTs > ts {
			minTs = ts
		}
		if maxTs < ts {
			maxTs = ts
		}
		totalTs += ts
		fmt.Printf("回覆自 %d.%d.%d.%d: 位元組=%d 時間=%dms TTL=%d \n", buf[12], buf[13], buf[14], buf[15], n-28, ts, buf[8])
		time.Sleep(time.Second)
	}
	fmt.Printf("%s 的 Ping 統計資料: 封包: 已傳送 = %d，已收到 = %d, 已遺失 = %d (%2.f%% 遺失)，大約的來回時間 (毫秒):\n 最小值 = %dms，最大值 = %dms，平均 = %dms",
		conn.RemoteAddr(), sendCount, successCount, failCount, float64(failCount)/float64(sendCount)*100, minTs, maxTs, totalTs/int64(successCount))

	// fmt.Println(buffer)
}

func getCommandArgs() {
	flag.Int64Var(&timeout, "w", 1000, "請求超時") //用flag 去接收64位元 timeout 參數 且w可以 -w
	flag.IntVar(&size, "l", 32, "請求發送緩衝區大小")
	flag.IntVar(&count, "n", 4, "發送請求數")
	flag.Parse() //解析參數
}

func checksum(data []byte) uint16 { // checksum 算法 將高位與低位相加 例如: 0x10 和 0x02 要變成 0x1002然後再入sum中 sum是32位元 再取反 得到checksum
	length := len(data)
	index := 0
	var sum uint32 = 0
	for length > 1 {
		sum += uint32(data[index])<<8 + uint32(data[index+1]) //計算checksum 相連兩位相加
		length -= 2
		index += 2
	}
	if length != 0 {
		sum += uint32(data[index]) //如果是基數個 最後一個孤獨者也是要加
	}
	hi16 := sum >> 16 //把低位拿掉 把高位提取出來
	for hi16 != 0 {
		sum = hi16 + uint32(uint16(sum)) //拿高位加上低位 再把結果存回sum
		hi16 = sum >> 16                 // 重複動作直到高位是0
	}
	return uint16(^sum) //再去從>>16之前的sum 取反 得到checksum
}
