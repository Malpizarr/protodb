package data

import (
	"fmt"

	"github.com/Malpizarr/dbproto/pkg/dbdata"
	"google.golang.org/protobuf/types/known/structpb"
)

type JoinType int

const (
	InnerJoin JoinType = iota
	LeftJoin
	RightJoin
	FullOuterJoin
)

// JoinTables Func to create joins  between two tables it perform one to many join between two tables, based on the key fields provided

func JoinTables(t1, t2 *Table, key1, key2 string, joinType JoinType) ([]map[string]interface{}, error) {
	results := make([]map[string]interface{}, 0)

	if err := t1.ResetAndLoadIndexes(); err != nil {
		return nil, fmt.Errorf("failed to load indexes for table 1: %v", err)
	}
	if err := t2.ResetAndLoadIndexes(); err != nil {
		return nil, fmt.Errorf("failed to load indexes for table 2: %v", err)
	}

	// Process records from t1
	for _, rec1 := range t1.Indexes[key1] {
		if rec1 == nil {
			continue
		}

		// Attempt to find matching records in t2
		matched := false
		for _, rec2 := range t2.Indexes[key2] {
			if rec2 != nil && Equal(rec1.Fields[key1], rec2.Fields[key2]) {
				results = append(results, mergeRecords(rec1, rec2))
				matched = true
			}
		}

		// If no match found and it's a left join or full outer join, add rec1 alone
		if !matched && (joinType == LeftJoin || joinType == FullOuterJoin) {
			results = append(results, mergeRecords(rec1, nil))
		}
	}

	// Process records from t2 if it's a right join or full outer join
	if joinType == RightJoin || joinType == FullOuterJoin {
		for _, rec2 := range t2.Indexes[key2] {
			if rec2 == nil {
				continue
			}

			// Check if rec2 was matched
			matched := false
			for _, rec1 := range t1.Indexes[key1] {
				if rec1 != nil && Equal(rec1.Fields[key1], rec2.Fields[key2]) {
					matched = true
					break
				}
			}

			// If no corresponding rec1 was found, add rec2 alone
			if !matched {
				results = append(results, mergeRecords(nil, rec2))
			}
		}
	}

	return results, nil
}

// mergeRecords merges two dbdata.Record objects and returns a map of field names to their corresponding values.
// The function extracts the values from the input records and prefixes the field names with "t1." or "t2."
// depending on the record they belong to.
func mergeRecords(rec1, rec2 *dbdata.Record) map[string]interface{} {
	result := make(map[string]interface{})
	// this will extract the value from the records and put it in the result map
	extractValue := func(v *structpb.Value) interface{} {
		switch x := v.Kind.(type) {
		case *structpb.Value_StringValue:
			return x.StringValue
		case *structpb.Value_NumberValue:
			return x.NumberValue
		case *structpb.Value_BoolValue:
			return x.BoolValue
		default:
			return nil
		}
	}

	if rec1 != nil {
		for k, v := range rec1.Fields {
			if v != nil {
				result["t1."+k] = extractValue(v)
			}
		}
	}
	if rec2 != nil {
		for k, v := range rec2.Fields {
			if v != nil {
				result["t2."+k] = extractValue(v)
			}
		}
	}
	return result
}
