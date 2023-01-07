package main

import (
	"errors"
	"flag"
	"os"
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"github.com/jackc/pgproto3/v2"

	"github.com/jackc/pgio"
	"github.com/wal-g/tracelog"
)

var readResp = flag.Bool("v", false, "Logs every packet in great detail")
var doErr = flag.Bool("e", false, "Logs every packet in great detail")

func getC(port int) (net.Conn, error) {
	const proto = "tcp"
	addr := "[::1]:" + strconv.Itoa(port)
	return net.Dial(proto, addr)
}

func readCnt(fr *pgproto3.Frontend, count int) error {
	for i := 0; i < count; i++ {
		if msg, err := fr.Receive(); err != nil {
			return err
		} else {
			tracelog.InfoLogger.Printf("received %T msg", msg)
		}
	}

	return nil
}

var okerr = errors.New("something")


type ConnState struct {
	fr *pgproto3.Frontend
	conn net.Conn
	ProcessID uint32
	SecretKey uint32
}

func (c *ConnState) waitRFQ() error {
	for {
		if msg, err := c.fr.Receive(); err != nil {
			return err
		} else {
			tracelog.InfoLogger.Printf("received %T %+v msg", msg, msg)
			switch q := msg.(type) {
			case *pgproto3.ParameterStatus:
				//ok
			case *pgproto3.BackendKeyData:
				c.ProcessID = q.ProcessID
				c.SecretKey = q.SecretKey
			case *pgproto3.ErrorResponse:
				return okerr
			case *pgproto3.ReadyForQuery:
				return nil
			}
		}
	}
}


func (c *ConnState) allocnewconn(port int) {
	conn, err := getC(port)
	if err != nil {
		tracelog.ErrorLogger.Printf("failed to get conn %w", err)
		if err != okerr {
			panic(err)
		}
		return
	}
	c.conn = conn
	c.fr = pgproto3.NewFrontend(pgproto3.NewChunkReader(conn), conn)

	if err := c.fr.Send(&pgproto3.StartupMessage{
		ProtocolVersion: 196608,
		Parameters: map[string]string{
			"user":     "reshke",
			"database": "yoba",
		},
	}); err != nil {
		tracelog.ErrorLogger.Printf("startup failed %w", err)
		if err != okerr {
			panic(err)
		}
	}

	if err := c.waitRFQ(); err != nil {
		tracelog.ErrorLogger.Printf("startup failed %w", err)
		if err != okerr {
			panic(err)
		}
		return
	}
}


const cancelRequestCode = 80877102

type CancelRequestMeta struct {
	ProcessID uint32
	SecretKey uint32
}

type CancelRequest struct {
	regs []CancelRequestMeta
}

// Encode encodes src into dst. dst will include the 4 byte message length.
func (src *CancelRequest) Encode(dst []byte) []byte {
	dst = pgio.AppendInt32(dst, int32(8 + len(src.regs) * 8))
	dst = pgio.AppendInt32(dst, cancelRequestCode)
	for i := 0; i < len(src.regs); i++ {
		dst = pgio.AppendUint32(dst, src.regs[i].ProcessID)
		dst = pgio.AppendUint32(dst, src.regs[i].SecretKey)
	}
	return dst
}


func gaogao(waitforres bool) error {
	state := map[string] *ConnState{}
	recordings := map[string][]string{}
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("hello, yoba@> ")

		text, err := reader.ReadString('\n')
		if err != nil {
			return nil;
		}

		text = strings.TrimSpace(text)

		fmt.Printf("recv command %s \n", text)
		if len(text) == 0 {
			continue
		}

		switch text {
		case "multiexecute":
			fmt.Printf("enter recording name: ")
			currcname, err := reader.ReadString('\n')
			if err != nil {
				return nil;
			}

			currcname = strings.TrimSpace(currcname)

			qconn, ok := state[currcname]
			if !ok {
				fmt.Printf("no such conn: %s\n", currcname)
				continue
			}
			fmt.Printf("enter rep times: ")

			cname, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			times, err := strconv.Atoi(cname)
			if err != nil {
				panic(err)
				return err
			}

			fmt.Printf("enter rep port: ")
			portStr, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			port, err := strconv.Atoi(portStr)
			if err != nil {
				panic(err)
				return err
			}

			// dont care of result

			for i := 0; i < times; i++ {
				go func () {
					qconn = &ConnState{}

					qconn.allocnewconn(port)
					for _, text := range  recordings[currcname] {
						if err := qconn.fr.Send(&pgproto3.Query{
							String: text,
						}); err != nil {
							panic(err)
						}
						if err := qconn.waitRFQ(); err != nil {
							tracelog.ErrorLogger.Printf("executing failed %w", err)
							if err != okerr {
								panic(err)
							}
							return
						}
					}
				} ()
			}

		case "record":
			fmt.Printf("enter recording name: ")
			currrname, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			currrname = strings.TrimSpace(currrname)
			lines := make([]string, 0)

			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				line = strings.TrimSpace(line)

				if line == "recend" {
					break
				}
				lines = append(lines, line)
			}

			recordings[currrname] = lines

		case "connect":
			fmt.Printf("enter connection name: ")
			cname, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			fmt.Printf("enter connection port: ")

			portStr, err := reader.ReadString('\n')
			if err != nil {
				panic(err)
				return err
			}

			portStr = strings.TrimSpace(portStr)

			port, err := strconv.Atoi(portStr)
			if err != nil {
				panic(err)
				return err
			}

			cname = strings.TrimSpace(cname)

			state[cname] = &ConnState{}

			state[cname].allocnewconn(port)
		case "query":
			fmt.Printf("enter target connection: ")
			currcname, err := reader.ReadString('\n')
			if err != nil {
				return nil;
			}

			currcname = strings.TrimSpace(currcname)

			qconn, ok := state[currcname]
			if !ok {
				fmt.Printf("no such conn: %s\n", currcname)
				continue
			}
			fmt.Printf("enter yout query@> ")

			text, err := reader.ReadString('\n')
			if err != nil {
				panic(err)
			}
			text = strings.TrimSpace(text)
			fmt.Printf("executing your query %s\n", text)
			if err := qconn.fr.Send(&pgproto3.Query{
				String: text,
			}); err != nil {
				panic(err)
			}

			if err := qconn.waitRFQ(); err != nil {
				tracelog.ErrorLogger.Printf("exeting failed %w", err)
				if err != okerr {
					panic(err)
				}
				return err
			}
		case "fire":
			/* and forget */

			fmt.Printf("enter target connection: ")
			currcname, err := reader.ReadString('\n')
			if err != nil {
				return nil;
			}

			currcname = strings.TrimSpace(currcname)

			qconn, ok := state[currcname]
			if !ok {
				fmt.Printf("no such conn: %s\n", currcname)
				continue
			}
			fmt.Printf("enter yout query@> ")

			text, err := reader.ReadString('\n')
			if err != nil {
				panic(err)
			}
			text = strings.TrimSpace(text)
			fmt.Printf("executing your query %s\n", text)
			if err := qconn.fr.Send(&pgproto3.Query{
				String: text,
			}); err != nil {
				panic(err)
			}
			// dont care of result

			go func () {
				if err := qconn.waitRFQ(); err != nil {
					tracelog.ErrorLogger.Printf("executing failed %w", err)
					if err != okerr {
						panic(err)
					}
					return
				}
			} ()

		case "shutdown all":
			cr := CancelRequest{
				regs: []CancelRequestMeta{},
			}
			for _, c := range state {
				fmt.Printf("%d %d \n", c.ProcessID, c.SecretKey);
				cr.regs = append(cr.regs, CancelRequestMeta{c.ProcessID, c.SecretKey})
			}

			conn, err := getC(5432)
			if err != nil {
				tracelog.ErrorLogger.Printf("failed to get conn %w", err)
				if err != okerr {
					panic(err)
				}
				return nil
			}

			if n, err := conn.Write(cr.Encode(nil)); err != nil {
				tracelog.ErrorLogger.Printf("startup failed %w", err)
				if err != okerr {
					panic(err)
				}
			} else {
				fmt.Printf("send %d\n", n)
			}
		default:
			fmt.Printf("fail!\n")
		}
	}

	tracelog.InfoLogger.Printf("ok")
	return nil
}

func main() {
	flag.Parse()

	gaogao(*readResp)
}
