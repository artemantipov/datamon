# Datamon
Monitoring data for different type of databases and provide results in prometheus metric format.

## Types of check:

### diff
Compare results in float64 , result value is discrepancy percentage source to destination as float64 with 2 decimals.

**Expected check query results:**
* one row with multiple columns (result metric - 1 metric for each column to column comparison)
* multiple row with one column (result metric - 1 metric as mean of all rows discrepancy percentage)
### compare
Compare results in string/varchar 1 to 1, result values is bool as int (0 - equal,1 - not equal).

**Expected check query results:**
* one row with multiple columns (result metric - 1 metric for each column to column comparison)

_Note: Column names should be the same for source and destination query response_

#### Result metrics examples

_Metrics for diff_

`datamon_check_metric{name="somecheck_column1", type="diff"} 13.37`

`datamon_check_metric{name="somecheck_column2", type="diff"} 3.14`

_Metrics for compare_

`datamon_check_metric{name="somecheck_somecolumn", type="compare"} 0`

`datamon_check_metric{name="somecheck_anothercolumn", type="compare"} 1`

### Supported db types:
* vertica
* postgres
* clickhouse
* mysql
