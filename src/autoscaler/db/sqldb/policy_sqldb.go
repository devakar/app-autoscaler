package sqldb

import (
	"database/sql"
	"encoding/json"

	"code.cloudfoundry.org/lager"
	_ "github.com/lib/pq"

	"autoscaler/db"
	"autoscaler/models"
	"fmt"
	"time"
)

type PolicySQLDB struct {
	url    string
	logger lager.Logger
	sqldb  *sql.DB
}

func NewPolicySQLDB(url string, logger lager.Logger) (*PolicySQLDB, error) {
	sqldb, err := sql.Open(db.PostgresDriverName, url)
	if err != nil {
		logger.Error("open-policy-db", err, lager.Data{"url": url})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		sqldb.Close()
		logger.Error("ping-policy-db", err, lager.Data{"url": url})
		return nil, err
	}

	return &PolicySQLDB{
		url:    url,
		logger: logger,
		sqldb:  sqldb,
	}, nil
}

func (pdb *PolicySQLDB) Close() error {
	err := pdb.sqldb.Close()
	if err != nil {
		pdb.logger.Error("Close-policy-db", err, lager.Data{"url": pdb.url})
		return err
	}
	return nil
}

func (pdb *PolicySQLDB) GetAppIds() (map[string]bool, error) {
	appIds := make(map[string]bool)
	query := "SELECT app_id FROM policy_json"

	rows, err := pdb.sqldb.Query(query)
	if err != nil {
		pdb.logger.Error("get-appids-from-policy-table", err, lager.Data{"query": query})
		return nil, err
	}
	defer rows.Close()

	var id string
	for rows.Next() {
		if err = rows.Scan(&id); err != nil {
			pdb.logger.Error("get-appids-scan", err)
			return nil, err
		}
		appIds[id] = true
	}
	return appIds, nil
}

func (pdb *PolicySQLDB) RetrievePolicies() ([]*models.PolicyJson, error) {
	query := "SELECT app_id,policy_json FROM policy_json WHERE 1=1 "
	policyList := []*models.PolicyJson{}
	rows, err := pdb.sqldb.Query(query)
	if err != nil {
		pdb.logger.Error("retrive-policy-list-from-policy_json-table", err,
			lager.Data{"query": query})
		return policyList, err
	}

	defer rows.Close()

	var appId string
	var policyStr string

	for rows.Next() {
		if err = rows.Scan(&appId, &policyStr); err != nil {
			pdb.logger.Error("scan-policy-from-search-result", err)
			return nil, err
		}
		policyJson := models.PolicyJson{
			AppId:     appId,
			PolicyStr: policyStr,
		}
		policyList = append(policyList, &policyJson)
	}
	return policyList, nil
}

func (pdb *PolicySQLDB) GetAppPolicy(appId string) (*models.ScalingPolicy, error) {
	var policyJson []byte
	query := "SELECT policy_json FROM policy_json WHERE app_id = $1"
	err := pdb.sqldb.QueryRow(query, appId).Scan(&policyJson)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		pdb.logger.Error("get-app-policy-from-policy-table", err, lager.Data{"query": query, "appid": appId})
		return nil, err
	}

	scalingPolicy := &models.ScalingPolicy{}
	err = json.Unmarshal(policyJson, scalingPolicy)
	if err != nil {
		pdb.logger.Error("get-app-policy-unmarshal", err, lager.Data{"policyJson": string(policyJson)})
		return nil, err
	}
	return scalingPolicy, nil
}

func (pdb *PolicySQLDB) FetchLock() (lock models.Lock, err error) {
	var (
		owner     string
		timestamp int64
		ttl       int
	)
	fetchLockErr := pdb.sqldb.QueryRow("SELECT * FROM locks").Scan(&owner, &timestamp, &ttl)
	switch {
	case fetchLockErr == sql.ErrNoRows:
		pdb.logger.Info("No lock entry found")
		return models.Lock{}, fetchLockErr
	case fetchLockErr != nil:
		pdb.logger.Error("Error occurs during lock fetching", err)
		return models.Lock{}, fetchLockErr
	default:
		pdb.logger.Info("Lock exist")
		lock := models.Lock{Owner: owner, Last_Modified_Timestamp: timestamp, Ttl: ttl}
		return lock, nil
	}
}

func (pdb *PolicySQLDB) ClaimLock(lockDetails models.Lock) (claimed bool, err error) {
	fmt.Println("No lock owner found! Lets claim the lock")
	tx, err := pdb.sqldb.Begin()
	pdb.logger.Info("Transaction started ")
	if err != nil {
		return false, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			pdb.logger.Error("Error during Rollback", err)
			return
		}
		pdb.logger.Info("Commiting Transaction..")
		err = tx.Commit()
		if err != nil {
			pdb.logger.Error("Error during commit", err)
		}
	}()
	pdb.logger.Info("waiting for SELECT for UPDATE")
	if _, err := tx.Exec("select * from locks FOR UPDATE"); err != nil {
		return false, err
	}
	pdb.logger.Info("Inserting the lock details")
	fmt.Println(lockDetails.Owner, lockDetails.Last_Modified_Timestamp, lockDetails.Ttl)
	query := "INSERT INTO locks (owner,lock_timestamp,ttl) VALUES ($1,$2,$3)"
	if _, err := tx.Exec(query, lockDetails.Owner, lockDetails.Last_Modified_Timestamp, lockDetails.Ttl); err != nil {
		return false, err
	}
	return true, nil
}

func (pdb *PolicySQLDB) RenewLock(owner string) error {
	fmt.Println("Lets renew my ownership")
	tx, err := pdb.sqldb.Begin()
	pdb.logger.Info("Transaction started ")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			pdb.logger.Error("Error during Rollback", err)
			return
		}
		pdb.logger.Info("Commiting Transaction..")
		err = tx.Commit()
		if err != nil {
			pdb.logger.Error("Error during commit", err)
		}
	}()
	pdb.logger.Info("waiting for SELECT for UPDATE")
	query := "SELECT * FROM locks where owner=$1 FOR UPDATE"
	if _, err := tx.Exec(query, owner); err != nil {
		return err
	}
	pdb.logger.Info("Renewing the lock details")
	currentTime := time.Now().Unix()
	updatequery := "UPDATE locks SET lock_timestamp=$1 where owner=$2"
	if _, err := tx.Exec(updatequery, currentTime, owner); err != nil {
		return err
	}
	return nil
}

func (pdb *PolicySQLDB) ReleaseLock(owner string) error {
	fmt.Println("I am done !,Lets release the lock")
	tx, err := pdb.sqldb.Begin()
	pdb.logger.Info("Transaction started ")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			pdb.logger.Error("Error during Rollback", err)
			return
		}
		pdb.logger.Info("Commiting Transaction..")
		err = tx.Commit()
		if err != nil {
			pdb.logger.Error("Error during commit", err)
		}
	}()
	pdb.logger.Info("waiting for SELECT for UPDATE")
	if _, err := tx.Exec("select * from locks FOR UPDATE"); err != nil {
		return err
	}
	pdb.logger.Info("Delete the lock details")
	query := "DELETE FROM locks WHERE owner = $1"
	if _, err := tx.Exec(query, owner); err != nil {
		return err
	}
	return nil
}

func (pdb *PolicySQLDB) AcquireLock(owner string, timestamp int64, ttl int) (bool, error) {

	fetchedLock, err := pdb.FetchLock()
	if err != nil && err == sql.ErrNoRows {
		fmt.Println("No lock owner found! Lets claim the lock")
		newLock := models.Lock{Owner: owner, Last_Modified_Timestamp: timestamp, Ttl: ttl}
		isClaimed, err := pdb.ClaimLock(newLock)
		if err != nil {
			fmt.Println(err)
			return false, err
		}
		fmt.Println("Is lock claimed?", isClaimed)
	} else if err != nil && err != sql.ErrNoRows {
		fmt.Println(err)
		return false, err
	} else {
		fmt.Println("Lock already owned by ", fetchedLock.Owner)
		if fetchedLock.Owner == owner {
			fmt.Println("I am the Owner!", owner)
			err := pdb.RenewLock(owner)
			if err != nil {
				fmt.Println(err)
				return false, err
			}
		} else {
			fmt.Println("Someone else is the Owner :", fetchedLock.Owner, " I am :", owner)
			lastUpdatedTime := time.Unix(fetchedLock.Last_Modified_Timestamp, 0)
			if lastUpdatedTime.Add(time.Second * time.Duration(fetchedLock.Ttl)).Before(time.Now()) {
				fmt.Println("Lock not renewed! Lets forcefully grab the lock")
				err := pdb.ReleaseLock(fetchedLock.Owner)
				if err != nil {
					pdb.logger.Error("Failed to release lock forcefully", err)
					return false, err
				}
				newLock := models.Lock{Owner: owner, Last_Modified_Timestamp: timestamp, Ttl: ttl}
				isClaimed, err := pdb.ClaimLock(newLock)
				if err != nil {
					fmt.Println(err)
					return false, err
				}
				fmt.Println("Is lock claimed?", isClaimed)
			} else {
				fmt.Println("Lock renewed and hold by owner")
				return false, nil
			}
		}
	}
	return true, nil
}
