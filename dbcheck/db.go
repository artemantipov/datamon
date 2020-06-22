package dbcheck

import (
	"datamon/config"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	_ "github.com/ClickHouse/clickhouse-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/vertica/vertica-sql-go"

	"github.com/jmoiron/sqlx"
)

var (
	datamonConf config.DatamonConfig
	queue       = make(chan [3]string, 1000)
)

func dbConnect(name string) (db *sqlx.DB) {
	conn := datamonConf.DB[name]
	var conStr string
	switch conn.Type {
	case "mysql":
		conStr = fmt.Sprintf("%v:%v@(%v:%v)/%v?charset=utf8&parseTime=True&loc=Local", conn.User, conn.Pass, conn.Host, conn.Port, conn.DB)
	case "postgres":
		conStr = fmt.Sprintf("host=%v port=%v user=%v dbname=%v sslmode=disable password=%v", conn.Host, conn.Port, conn.User, conn.DB, conn.Pass)
	case "vertica":
		conStr = fmt.Sprintf("vertica://%v:%v@%v:%v/%v", conn.User, conn.Pass, conn.Host, conn.Port, conn.DB)
	case "clickhouse":
		conStr = fmt.Sprintf("tcp://%v:%v?username=%v&password=%v&database=%v", conn.Host, conn.Port, conn.User, conn.Pass, conn.DB)
	default:
		fmt.Println("DB type required - mysql|postgres|vertica|clickhouse")
	}
	db, err := sqlx.Connect(conn.Type, conStr)
	if err != nil {
		log.Printf("Connection Failed to Open for %v with ERROR: %v", name, err)
	} else {
		log.Printf("Connection Established for %v", name)
	}
	return
}

//ScheduleCheck run check go routine with interval
func scheduleCheck(name string, check config.DbCheckStruct) {
	for {
		interval, err := strconv.Atoi(check.Interval)
		if err != nil {
			log.Fatalf("Failed to parse interval value: %v with error: %v", check.Interval, err)
		}
		runCheck(name, check)
		time.Sleep(time.Duration(interval) * time.Minute)
	}
}

func runCheck(name string, check config.DbCheckStruct) {
	switch check.CheckType {
	case "diff":
		diffCheck(name, check)
	case "compare":
		compareCheck(name, check)
	default:
		log.Println("check type required")
	}
}

//Start run db checks
func Start() {
	datamonConf = config.ReadConfig()
	for name, check := range datamonConf.Checks {
		go scheduleCheck(name, check)
	}
}

//GetQueue return metrics queue
func GetQueue() chan [3]string {
	return queue
}

func dbClose(name string, db *sqlx.DB) {
	err := db.Close()
	if err != nil {
		log.Printf("Error during close connect to %v", name)
	} else {
		log.Printf("Connection closed for %v", name)
	}
}

func compareCheck(name string, check config.DbCheckStruct) {
	log.Printf("Start COMPARE check for %v", name)
	compares := map[string]float64{}
	srcDB := dbConnect(check.Src)
	dstDB := dbConnect(check.Dst)
	defer dbClose(check.Src, srcDB)
	defer dbClose(check.Dst, dstDB)
	srcQuery, dstQuery := check.QueryStrings()
	srcResults := dbQuery(srcDB, srcQuery)
	dstResults := dbQuery(dstDB, dstQuery)
	if len(srcResults) != len(dstResults) {
		log.Printf("Columns count mismatch for COMPARE %v: SRC:%v DST:%v", name, len(srcResults), len(dstResults))
	} else {
		for i, _ := range srcResults {
			srcValues := srcResults[i]
			dstValues := dstResults[i]
			for colName, _ := range srcValues {
				if len(srcValues[colName]) != 1 || len(dstValues[colName]) != 1 {
					log.Printf("Columns response count more thet 1 for COMPARE %v: Column: %v SRC:%v DST:%v", name, colName, len(srcValues[colName]), len(dstValues[colName]))
				} else {
					for col, _ := range srcResults[0] {
						if srcResults[0][col][0] == dstResults[0][col][0] {
							compares[col] = 0
						} else {
							compares[col] = 1
						}
					}
				}
			}
		}
	}
	for k, v := range compares {
		metric := [3]string{fmt.Sprintf("%v_%v", name, k), check.CheckType, fmt.Sprintf("%v", v)}
		log.Printf("%+v", metric)
		queue <- metric
	}
}

func diffCheck(name string, check config.DbCheckStruct) {
	log.Printf("Start DIFF check %v", name)
	diffs := map[string]float64{}
	srcDB := dbConnect(check.Src)
	dstDB := dbConnect(check.Dst)
	defer dbClose(check.Src, srcDB)
	defer dbClose(check.Dst, dstDB)
	//srcQuery := check.Query.Src
	//dstQuery := check.Query.Dst
	srcQuery, dstQuery := check.QueryStrings()
	srcResults := dbQuery(srcDB, srcQuery)
	dstResults := dbQuery(dstDB, dstQuery)
	if len(srcResults) != len(dstResults) {
		log.Printf("Columns count mismatch for DIFF %v: SRC:%v DST:%v", name, len(srcResults), len(dstResults))
	} else {
		for i, _ := range srcResults {
			srcValues := srcResults[i]
			dstValues := dstResults[i]
			for colName, _ := range srcValues {
				if len(srcValues[colName]) != len(dstValues[colName]) {
					log.Printf("Columns response count mismatch for DIFF %v: Column: %v SRC:%v DST:%v", name, colName, len(srcValues[colName]), len(dstValues[colName]))
				} else {
					var tmpDiffs []float64
					for n, _ := range srcValues[colName] {
						srcValue, err := strconv.ParseFloat(srcValues[colName][n], 64)
						if err != nil {
							log.Printf("Failed parse int for %v", srcValues[colName][n])
						}
						dstValue, err := strconv.ParseFloat(dstValues[colName][n], 64)
						if err != nil {
							log.Printf("Failed parse int for %v", srcValues[colName][n])
						}
						var diff float64
						if srcValue == 0 && dstValue != srcValue {
							diff = 100
						} else if dstValue == 0 && srcValue != dstValue {
							diff = 100
						} else if srcValue == dstValue && srcValue == 0 {
							diff = 0
						} else {
							delta := srcValue - dstValue
							diff = math.Abs(delta / srcValue * 100)
						}
						diffCut, err := strconv.ParseFloat(fmt.Sprintf("%.2f", diff), 64)
						if err != nil {
							log.Printf("Failed to short decimals for float64 %v", diff)
						}
						tmpDiffs = append(tmpDiffs, diffCut)
					}
					diffResult := calcMeanMedian(tmpDiffs)
					diffs[colName] = diffResult
				}
			}
		}
	}
	for k, v := range diffs {
		metric := [3]string{fmt.Sprintf("%v_%v", name, k), check.CheckType, fmt.Sprintf("%v", v)}
		log.Printf("%+v", metric)
		queue <- metric
	}
}

func calcMeanMedian(floatSlice []float64) float64 {
	//sort.Float64s(floatSlice)
	//mNumber := len(floatSlice) / 2
	//if len(floatSlice)%2 == 0 {
	//	fmt.Printf("MEDIAN FOR %v is %v\n",floatSlice, floatSlice[mNumber])
	//	return floatSlice[mNumber]
	//}
	//fmt.Printf("MEDIAN FOR %v is %v\n",floatSlice, (floatSlice[mNumber-1] + floatSlice[mNumber]) / 2)
	//return (floatSlice[mNumber-1] + floatSlice[mNumber]) / 2
	var sum float64
	for i, _ := range floatSlice {
		sum += floatSlice[i]
	}
	length, err := strconv.ParseFloat(fmt.Sprintf("%v", len(floatSlice)), 64)
	if err != nil {
		log.Printf("Failed parse float64 for %v", len(floatSlice))
	}
	mean, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", sum/length), 64)
	if err != nil {
		log.Printf("Failed to short decimals for %v", sum/length)
	}
	return mean
}

func dbQuery(db *sqlx.DB, query string) (results []map[string][]string) {
	rows, err := db.Queryx(query)
	if err != nil {
		log.Printf("Failed to query with error: %v", err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		log.Printf("Failed to get columns with error: %v", err)
	}
	result := make(map[string][]string)
	if len(columns) > 1 {
		for rows.Next() {
			tmpResult := make(map[string]interface{})
			err = rows.MapScan(tmpResult)
			if err != nil {
				log.Printf("Failed to map columns with error: %v", err)
			}
			for col, val := range tmpResult {
				if v, ok := val.([]byte); ok {
					val = string(v)
				}
				result[col] = []string{fmt.Sprintf("%v", val)}
			}
			results = append(results, result)
		}
	} else {
		var colSlice []string
		for rows.Next() {
			var row string
			err = rows.Scan(&row)
			if err != nil {
				log.Printf("Failed to scan rows with err: %v", err)
			}
			colSlice = append(colSlice, row)
		}
		result[columns[0]] = colSlice
		results = append(results, result)
	}
	return
}
