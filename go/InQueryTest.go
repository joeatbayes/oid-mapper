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
)
 
const (
    host        = "localhost"
    port        = 5432
    defaultUser = "postgres"
    defaultPass = "test"
    dbname      = "oidmap"
)

func CheckError(err error) {
    if err != nil {
        panic(err)
    }
}
 
func main() {
    // connection string
    osuser := os.Getenv("PGUSER");
    ospass := os.Getenv("PGPASS");
    if osuser < " " { osuser =  defaultUser }
    if ospass < " " { ospass =  defaultPass }
    
    psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, osuser, ospass, dbname)
    
        // open database
    db, err := sql.Open("postgres", psqlconn)
    CheckError(err)
     
    
    // check db
    err = db.Ping()
    CheckError(err)
    fmt.Println("Connected!")
    fmt.Println("L41: err=", err)
    defer db.Close()
    
    // NOTE: This will fail to return records until updated with 
    // oids that match those in the current source file used to load 
    // the database such as ../test.map.txt
    rows, err := db.Query(`SELECT DISTINCT paroid, partbl FROM omap WHERE omap.chiloid IN ( '01f2ae1d-940d-4c8d-9c4c-b98c5337d4ee', '1c87fa56-a127-4712-a29a-165840d472e1', '14bdfcc6-e4ee-4786-ba20-e5ac42da204f', '88fa63a5-0a44-430e-8a3f-96ee65636bc3', '5f64e91b-de6b-4f16-9866-790954e8f2af'
    )`)
    CheckError(err)
    fmt.Println("L45: err=", err)
 
    defer rows.Close()
    
    for rows.Next() {
       var paroid, partbl string       
       err = rows.Scan(&paroid, &partbl)
       CheckError(err)
       fmt.Println("parId=", paroid, partbl)
    }
     
 
   
}
 
 
