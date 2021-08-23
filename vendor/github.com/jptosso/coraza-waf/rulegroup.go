// Copyright 2021 Juan Pablo Tosso
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package engine

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/jptosso/coraza-waf/utils"
	"go.uber.org/zap"
)

type RuleGroup struct {
	rules []*Rule
	mux   *sync.RWMutex
}

// Adds a rule to the collection
// Will return an error if the ID is already used
func (rg *RuleGroup) Add(rule *Rule) error {
	if rule == nil {
		// this is an ugly solution but chains should not return rules
		return nil
	}

	if rg.FindById(rule.Id) != nil && rule.Id != 0 {
		return fmt.Errorf("there is a another rule with id %d", rule.Id)
	}
	rg.rules = append(rg.rules, rule)
	return nil
}

// GetRules returns the slice of rules,
// it's concurrent safe.
func (rg *RuleGroup) GetRules() []*Rule {
	rg.mux.RLock()
	defer rg.mux.RUnlock()
	return rg.rules
}

// FindById return a Rule with the requested Id
func (rg *RuleGroup) FindById(id int) *Rule {
	for _, r := range rg.rules {
		if r.Id == id {
			return r
		}
	}
	return nil
}

// DeleteById removes a rule by it's Id
func (rg *RuleGroup) DeleteById(id int) {
	for i, r := range rg.rules {
		if r != nil && r.Id == id {
			copy(rg.rules[i:], rg.rules[i+1:])
			rg.rules[len(rg.rules)-1] = nil
			rg.rules = rg.rules[:len(rg.rules)-1]
		}
	}
}

// FindByMsg returns a slice of rules that matches the msg
func (rg *RuleGroup) FindByMsg(msg string) []*Rule {
	rules := []*Rule{}
	for _, r := range rg.rules {
		if r.Msg == msg {
			rules = append(rules, r)
		}
	}
	return rules
}

// FindByTag returns a slice of rules that matches the tag
func (rg *RuleGroup) FindByTag(tag string) []*Rule {
	rules := []*Rule{}
	for _, r := range rg.rules {
		if utils.StringInSlice(tag, r.Tags) {
			rules = append(rules, r)
		}
	}
	return rules
}

// Count returns the count of rules
func (rg *RuleGroup) Count() int {
	return len(rg.rules)
}

// Clear will remove each and every rule stored
func (rg *RuleGroup) Clear() {
	rg.rules = []*Rule{}
}

// Evaluate rules for the specified phase, between 1 and 5
// Returns true if transaction is disrupted
func (rg *RuleGroup) Evaluate(phase int, tx *Transaction) bool {
	tx.Waf.Logger.Debug("transaction evaluated",
		zap.String("id", tx.Id),
		zap.Int("phase", phase),
	)
	tx.LastPhase = phase
	ts := time.Now().UnixNano()
	usedRules := 0
	tx.LastPhase = phase
	for _, r := range tx.Waf.Rules.GetRules() {
		if tx.Interruption != nil {
			return true
		}
		// Rules with phase 0 will always run
		if r.Phase != phase && r.Phase != 0 {
			continue
		}
		rid := strconv.Itoa(r.Id)
		if r.Id == 0 {
			rid = strconv.Itoa(r.ParentId)
		}
		if utils.IntInSlice(r.Id, tx.RuleRemoveById) {
			continue
		}
		//we always evaluate secmarkers
		if tx.SkipAfter != "" {
			if r.SecMark == tx.SkipAfter {
				tx.SkipAfter = ""
			}
			continue
		}
		if tx.Skip > 0 {
			tx.Skip--
			//Skipping rule
			continue
		}
		txr := tx.GetCollection(VARIABLE_RULE)
		txr.Set("id", []string{rid})
		txr.Set("rev", []string{r.Rev})
		severity := strconv.Itoa(r.Severity)
		txr.Set("severity", []string{severity})
		//txr.Set("logdata", []string{r.LogData})
		txr.Set("msg", []string{r.Msg})
		r.Evaluate(tx)

		tx.Capture = false //we reset the capture flag on every run
		usedRules++
	}
	tx.StopWatches[phase] = int(time.Now().UnixNano() - ts)
	return tx.Interruption != nil
}

func NewRuleGroup() *RuleGroup {
	return &RuleGroup{
		rules: []*Rule{},
		mux:   &sync.RWMutex{},
	}
}
