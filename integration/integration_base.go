/*
 Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.

 Licensed under the Apache License, Version 2.0 (the "License").
 You may not use this file except in compliance with the License.
 A copy of the License is located at

 http://www.apache.org/licenses/LICENSE-2.0

 or in the "license" file accompanying this file. This file is distributed
 on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
 express or implied. See the License for the specific language governing
 permissions and limitations under the License.
*/

package integration

import (
	"fmt"
	"qldbdriver/qldbdriver"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/qldb"
	"github.com/aws/aws-sdk-go/service/qldbsession"
	"github.com/stretchr/testify/assert"
)

type testBase struct {
	qldb       *qldb.QLDB
	ledgerName *string
	regionName *string
	logger     *testLogger
}

const (
	ledger 				   = "Gotest"
	region 				   = "us-east-1"
	testTableName          = "GoIntegrationTestTable"
	indexAttribute         = "Name"
	columnName             = "Name"
	singleDocumentValue    = "SingleDocumentValue"
	multipleDocumentValue1 = "MultipleDocumentValue1"
	multipleDocumentValue2 = "MultipleDocumentValue2"
)


func createTestBase() *testBase {
	sess, err := session.NewSession(aws.NewConfig().WithRegion(region))
	mySession := session.Must(sess, err)
	qldb := qldb.New(mySession)
	logger := &testLogger{&defaultLogger{}, LogInfo}
	ledgerName := ledger
	regionName := region
	return &testBase{qldb, &ledgerName, &regionName, logger}
}

func (testBase *testBase) createLedger(t *testing.T) {
	testBase.logger.log(fmt.Sprint("Creating ledger named ", *testBase.ledgerName, " ..."), LogInfo)
	deletionProtection := false
	permissions := "ALLOW_ALL"
	_, err := testBase.qldb.CreateLedger(&qldb.CreateLedgerInput{Name: testBase.ledgerName, DeletionProtection: &deletionProtection, PermissionsMode: &permissions})
	assert.Nil(t, err)
	testBase.waitForActive()
}

func (testBase *testBase) deleteLedger(t *testing.T) {
	testBase.logger.log(fmt.Sprint("Deleting ledger ", *testBase.ledgerName), LogInfo)
	deletionProtection := false
	testBase.qldb.UpdateLedger(&qldb.UpdateLedgerInput{DeletionProtection: &deletionProtection, Name: testBase.ledgerName})
	_, err := testBase.qldb.DeleteLedger(&qldb.DeleteLedgerInput{Name: testBase.ledgerName})
	if err != nil {
		if _, ok := err.(*qldb.ResourceNotFoundException); ok {
			testBase.logger.log("Encountered resource not found", LogInfo)
			return
		}
		testBase.logger.log("Encountered error during deletion", LogInfo)
		testBase.logger.log(err.Error(), LogInfo)
		t.Errorf("Failing test due to deletion failure")
	}
	testBase.waitForDeletion()
}

func (testBase *testBase) waitForActive() {
	testBase.logger.log("Waiting for ledger to become active...", LogInfo)
	for true {
		output, _ := testBase.qldb.DescribeLedger(&qldb.DescribeLedgerInput{Name: testBase.ledgerName})
		if *output.State == "ACTIVE" {
			testBase.logger.log("Success. Ledger is active and ready to use.", LogInfo)
			return
		}
		testBase.logger.log("The ledger is still creating. Please wait...", LogInfo)
		time.Sleep(5 * time.Second)
	}
}

func (testBase *testBase) waitForDeletion() {
	testBase.logger.log("Waiting for ledger to be deleted...", LogInfo)
	for true {
		_, err := testBase.qldb.DescribeLedger(&qldb.DescribeLedgerInput{Name: testBase.ledgerName})
		testBase.logger.log("The ledger is still deleting. Please wait...", LogInfo)
		if err != nil {
			if _, ok := err.(*qldb.ResourceNotFoundException); ok {
				testBase.logger.log("The ledger is deleted", LogInfo)
				return
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func (testBase *testBase) getDriver(ledgerName string, maxConcurrentTransactions uint16, retryLimit uint8) *qldbdriver.QLDBDriver {
	driverSession := session.Must(session.NewSession(aws.NewConfig().WithRegion(*testBase.regionName)))
	qldbsession := qldbsession.New(driverSession)
	return qldbdriver.New(ledgerName, qldbsession, func(options *qldbdriver.DriverOptions) {
		options.Logger = testBase.logger.logger
		options.LoggerVerbosity = qldbdriver.LogInfo
		options.MaxConcurrentTransactions = maxConcurrentTransactions
	})

}
