// in_query_test.go
// run setGOEnv.bat or manually set GOPATH from current directory first.
// run go get github.com/lib/pq to download the postgres driver.
// Run this from a shell where the pg_env has not been set or it will generate an error.

package main
 
import (
    "database/sql"
    "fmt"
    _ "github.com/lib/pq"
    "os"
    "io"
    "log"
    "bufio"
    "strings"
    "time"
)
 
const (
    host            = "127.0.0.1"
    port             = 5432
    defaultUser  = "test"
    defaultPass = "test"
    dbname       = "oidmap"
)

func CheckError(err error) {
    if err != nil {
        panic(err)
    }
}

func currMS() float64 {
        return  float64(time.Now().UnixNano()) / 1000000.0
}
 
// TODO: Add a multi-threaded buffer queue like I used in httpTest
// to allow exercising postgres with multiple concurent requests. 

func main() {
    maxBufOids := 50
    
    osuser := os.Getenv("PGUSER");
    ospass := os.Getenv("PGPASS");
    if osuser < " " { osuser =defaultUser }
    if ospass < " " { ospass =  defaultPass }
    
    psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, osuser, ospass, dbname)
    fmt.Println("L50: psqlconn=", psqlconn)
        // open database
    db, err := sql.Open("postgres", psqlconn)
    CheckError(err)
     
    
    // check db
    err = db.Ping()
    CheckError(err)
    fmt.Println("Connected!")
    fmt.Println("L41: err=", err)
    
   cliOids := []string{}
   f, err := os.Open("../test.map.txt")
   if err != nil {
    fmt.Println("error opening file ", err)
    os.Exit(1)
   }
   defer f.Close()
   r := bufio.NewReader(f)
   for {
      aline, err := r.ReadString(10) // 0x0A separator = newline
      if ((err == io.EOF) || (len(cliOids) >= maxBufOids)) {
        if len(cliOids) > 0 {
          // flush our accumulated buffer 
          //fmt.Println("L62: cliOids=", cliOids)
          oidsStr := strings.Join(cliOids, ", ")
          sqlStr := `SELECT DISTINCT paroid, partbl FROM omap WHERE omap.chiloid IN ( ` + oidsStr +  ` )`
          //fmt.Println("L64: sqlStr=", sqlStr)
          sqlStart := time.Now()
          rows, err := db.Query(sqlStr)
          sqlExecElap := time.Since(sqlStart)
          sqlIterStart:= time.Now()
    
          CheckError(err)
          //fmt.Println("L66: err=", err)
          defer rows.Close()
          for rows.Next() {
            var paroid, partbl string       
            err = rows.Scan(&paroid, &partbl)
            CheckError(err)
            fmt.Println("parId=", paroid, partbl)
          }
          sqlIterElap := time.Since(sqlIterStart)
          fmt.Println("SQL Exect=" , sqlExecElap, "SQL Iter=", sqlIterElap)
        }
        // Clear buffer for next pass
        cliOids = make([]string,0)
        if err == io.EOF {
           break;
        }
      } else if err != nil {
          log.Fatalf("read file line error: %v", err)
          return
      } else {
        // process the line and add client oid to buffer 
        //fmt.Println("aline=", aline);
        flds := strings.Split(strings.TrimSpace(aline),",")
        //fmt.Println("L86: flds=", flds)
        if len(flds) == 4 {
          cliOid := flds[3]
          cliOidStr := "'" + cliOid + "'"
          //fmt.Println("L91: cliOid=", cliOid, "cliOidStr=", cliOidStr)
          cliOids = append(cliOids, cliOidStr)
        }
      }
    }
}
 
 