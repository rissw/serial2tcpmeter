package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/tarm/serial"
)

func main() {
	log.Println("Start Program")

	connCount := 0
	li, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Println("Error TCP Server:", err)
	}
	defer li.Close()

	for {

		if connCount == 0 {
			cl, err := li.Accept()
			if err != nil {
				log.Println("Error client accept:", err)
			}
			connCount++
			log.Println("Accept client:", cl.RemoteAddr())

			port := "/dev/ttyUSB0"
			baud := 9600
			c := &serial.Config{Name: port, Baud: baud, ReadTimeout: 1 * time.Second}
			s, err := serial.OpenPort(c)
			if err != nil {
				log.Fatal(err)
			}
			frameLength := 0
			go func() {
				data := make([]byte, 0, 1024)
				buf := make([]byte, 32)
				pos := 0
				state := stInit
			MAINLOOP:
				for {
					n, err := s.Read(buf)
					if err != nil {
						if err == io.EOF {
							continue
						}
						break
					}
					if n != 0 {
						data = append(data, buf[:n]...)
					}

					for {
						if pos >= len(data) {
							continue MAINLOOP
						}

						switch state {
						case stInit:
							if data[pos] == 0x7e {
								if pos > 0 {
									copy(data, data[pos:])
									data = data[:len(data)-pos]
								}
								state = stA0A8
								pos = 1
								continue
							}
							pos++
						case stA0A8:
							if data[pos]&0xA0 == 0xA0 {
								state = stLen
							} else {
								state = stInit
							}
							pos++
						case stLen:
							state = stLenTo7E
							frameLength = int(data[pos])
						case stLenTo7E:
							if len(data) < frameLength+1 {
								continue MAINLOOP
							}
							state = stEnd7E
							pos = frameLength + 1
						case stEnd7E:
							if data[pos] == 0x7e {
								_, err := cl.Write(data[:pos+1])
								if err != nil {
									break MAINLOOP
								}
							}
							fmt.Printf("Serial TX: % 02x\n", data[:pos+1])

							data = data[:0]
							pos = 0
							state = stInit
							continue MAINLOOP
						}

					}

				}
				fmt.Println("GoRoutine ended")
			}()

			buf := make([]byte, 256)
			for {
				n, err := cl.Read(buf)
				if err != nil {
					break
				}

				fmt.Printf("Serial RX: % 02x\n", buf[:n])
				_, err = s.Write(buf[:n])
				if err != nil {
					break
				}
			}

			fmt.Println("Close serial")
			s.Close()
			fmt.Println("Close conn")
			cl.Close()

			connCount--

		} else {
			cl, err := li.Accept()
			if err == nil {
				cl.Close()
			}

		}
	}

	log.Println("End Program")
}

const (
	stInit byte = iota
	stStart7E
	stA0A8
	stLen
	stLenTo7E
	stEnd7E
)
