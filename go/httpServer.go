package main

//"encoding/json"

import (
    "fmt"
    //"io"
    //"io/ioutil"
    "log"
    "net/http"
    "net/url"
    "os"
    //"os/exec"
    "strings"
    "database/sql"
   _ "github.com/lib/pq"
    "io/ioutil"
    "time"
        
)

const (
    host            = "localhost"
    port             = 5432
    defaultUser = "postgres"
    defaultPass = "test"
    dbname      = "oidmap"
)


func CheckError(err error) {
    if err != nil {
        panic(err)
    }
}

func currMS() float64 {
        return  float64(time.Now().UnixNano()) / 1000000.0
}

// FileExists reports whether the named file exists as a boolean
func FileExists(name string) bool {
    if fi, err := os.Stat(name); err == nil {
        if fi.Mode().IsRegular() {
            return true
        }
    }
    return false
}

// DirExists reports whether the dir exists as a boolean
func DirExists(name string) bool {
    if fi, err := os.Stat(name); err == nil {
        if fi.Mode().IsDir() {
            return true
        }
    }
    return false
}

func quote(astr string) string {
  // TODO: Add some escaping to make SQL safe.
  astr = strings.ReplaceAll(astr, ";", "-");
  astr = strings.ReplaceAll(astr, "\"", "-");
  astr = strings.ReplaceAll(astr, "'", "-");
  astr = strings.ReplaceAll(astr, "\n", "-");
  astr = strings.ReplaceAll(astr, "(", "-");
  astr = strings.ReplaceAll(astr, ")", "-");
  astr = strings.ReplaceAll(astr, ":", "-");
  astr = strings.ReplaceAll(astr, "[", "-");
  astr = strings.ReplaceAll(astr, "]", "-");
  astr = strings.TrimSpace(astr);
  return "'" + astr + "'";
}


var db *sql.DB

/* Setup the basic SQL connection so we do not pay the overhead 
 during every call */
func bdInit() {
    var err error
    osuser := os.Getenv("PGUSER");
    ospass := os.Getenv("PGPASS");
    if osuser < " " { osuser =defaultUser }
    if ospass < " " { ospass =  defaultPass }

    //-----
    //-- Obtain SQL Connection
    //-----
    getConStart := time.Now()
    //---
    // Check the DB for requested oids
    //----
    psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, osuser, ospass, dbname)
         
         
    // open database
    // TODO: Move DB to a global variable and re-use
    db, err = sql.Open("postgres", psqlconn)
    CheckError(err)
    
    // check db
    err = db.Ping()
    CheckError(err)
    db.SetMaxIdleConns(25)
    getConElap := time.Since(getConStart)
    fmt.Println("Get DB Connection ", psqlconn, " elap=", getConElap)
}


/* Simple handler to demonstrate responding to
a posted form.   Can be accessed via:
http://127.0.0.1:9601/ping?name=Joe&handle=workinghard
values will print to console. */
func ping(w http.ResponseWriter, r *http.Request) {
    w.Header().Add("content-type", "text/plain")
    r.ParseForm() // parse arguments, you have to call this by yourself
    w.WriteHeader(http.StatusOK)
    fmt.Println(r.Form) // print form information in server side
    fmt.Println("path", r.URL.Path)
    fmt.Println("scheme", r.URL.Scheme)
    fmt.Println("query", r.URL.Query())
    fmt.Println("url ", r.URL.RequestURI())
    fmt.Println(r.Form["url_long"])
    fmt.Fprintf(w, "printFormVal\n url=%s\n", r.URL.Path) // send data to client side
    for k, v := range r.Form {
        fmt.Println("key:", k, "\t", " val:", strings.Join(v, ""))
        fmt.Fprintf(w, "key:%s\t val=%s\n", k, strings.Join(v, ""))
    }
    
    // Read the body string.
    fmt.Println("contentLength=", r.ContentLength)
    bodya := make([]byte, r.ContentLength)
    _, err := r.Body.Read(bodya)
    fmt.Println("read body err=", err)
    bodys := string(bodya)    
    fmt.Println("ready post bodys=", bodys, " err=", err)

    // Parse the Body String and compare the
    // and pull out the form ID with ID supplied
    // in the JSON body.
    //body := string(p)
    //fmt.Println("body as str=", body)
    //r.GetBody()
    //fmt.Println("printf r.body=", r.Body)
    //fmt.Println("urstr", urstr)
}


/* http Hander for /oidmap.   
  TODO:  Check to make sure we can re-use the same
  DB connection when running multiple 
  concurrent requets */
func oid_search(w http.ResponseWriter, r *http.Request) {
    reqStart := time.Now()
    w.Header().Add("content-type", "text/plain")
    //fmt.Println("ex_handle_query_parms", r.URL.Scheme)
    var urstr = r.URL.RequestURI()
    //fmt.Println("URL=", r.URL)

    // USE This approach to parse URL Query Paramters
    // as ?id=1004
    var ursplt = strings.SplitN(urstr, "?", 2)
    //fmt.Println("ursplt=", ursplt)
    if len(ursplt) < 2 {
        w.WriteHeader(http.StatusBadRequest)
        fmt.Fprintf(w, "query paramter id is mandatory")
        return
    }
    qparms, err := url.ParseQuery(ursplt[1])
    //fmt.Println(" parsedQuery=", qparms, " err=", err)
    tmpKeys, ok := qparms["keys"]
    keysStr := strings.Join(tmpKeys, "");
    //fmt.Println("keysStr=", keysStr);
    keys := strings.Split(keysStr, ",");
    bodyArr, bodyErr := ioutil.ReadAll(r.Body);
    r.Body.Close()
    if bodyErr != nil {
       fmt.Printf("Error reading body err=", bodyErr)
    }
    if len(bodyArr) > 0 {
       fmt.Printf("bytes read=", len(bodyArr))
    }
    //fmt.Println("keys=", keys);
    if !ok {
        w.WriteHeader(http.StatusBadRequest)
        fmt.Fprintf(w, "query paramter ?keys is mandatory")
        return
    }
    cliOids := []string{}
    for i,key := range(keys) {
      cliOids = append(cliOids, quote(key));
      if (i > 500) {
        break;
      }
    }
    oidsStr := strings.Join(cliOids, ", ")
    //fmt.Println("oidsStr=", oidsStr)
    parseReqElap := time.Since(reqStart)
    
    
    //-----
    //-- Process SQL Search
    //---- 
    // TODO: Buffer rows and return after we can count the total 
    // results.    
    sqlStr := `SELECT DISTINCT paroid, partbl FROM omap WHERE omap.chiloid IN ( ` + oidsStr +  ` )`
    //fmt.Println("L64: sqlStr=", sqlStr)
    sqlStart := time.Now()
    rows, err := db.Query(sqlStr)
    sqlExecElap := time.Since(sqlStart)
    sqlIterStart:= time.Now()
    CheckError(err)
    //fmt.Println("L66: err=", err)
    rows.Close()
    w.WriteHeader(http.StatusOK)
    cnt := 0;
    for rows.Next() {
       var paroid, partbl string       
       err = rows.Scan(&paroid, &partbl)
       CheckError(err)
       //fmt.Println("parId=", paroid, partbl)
       fmt.Fprintf(w, "%s,%s\n", partbl, paroid)
       cnt ++;
    }
    if cnt < 1 {
      fmt.Fprintf(w, "--ERROR NO RECORDS MATCHED\n")
    }
    totReqElap := time.Since(reqStart);
    sqlIterElap := time.Since(sqlIterStart)
    fmt.Fprintf(w,"--Num keys=%d\n", len(keys));
    fmt.Fprintf(w,"--Num Match=%d\n" , cnt);
    fmt.Fprintf(w,"--parse Req=%s\n", parseReqElap);
    fmt.Fprintf(w,"--SQL Exec=%s\n" , sqlExecElap);
    fmt.Fprintf(w,"--SQL Iter=%s\n", sqlIterElap);
    fmt.Fprintf(w,"--Tot Req Time=%s\n", totReqElap);
        
} // func


func main() {
    http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100
    bdInit()
    http.Handle("/", http.FileServer(http.Dir("../http-docs")))
    // When path ends with "/" it is treated as a tree root
    // which allos the handler to pick up the path and any
    // sub paths such as /ping/apple.
    fmt.Println("Listening on port 9832")
    cwd, _ := os.Getwd()
    fmt.Println("cwd=", cwd) // for example /home/user

    http.HandleFunc("/ping/", ping) // set router
    http.HandleFunc("/oidmap", oid_search) // set router
    //http.HandleFunc("/api/save-cert-of-need/", api_save_cert_of_need)
    // Test:  http://127.0.0.1:9832/api/oidmap?keys=101818,1818181,1171717
    //  should return a 200
    err := http.ListenAndServe(":9832", nil) // set listen port
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }

}
