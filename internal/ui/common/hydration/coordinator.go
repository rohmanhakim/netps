/* Package hydration
	- provides a coordinator pattern for managing
	- asynchronous data hydration across multiple fields.

	- The coordinator separates state management from data storage,
	allowing screens to focus on business logic while the coordinator
	handles aggregate state tracking, error management, and retry logic.

	- Usage:
    	coordinator := hydration.NewCoordinator()
    	coordinator.Register("field1", RarityCommon, true, true)

    	field := coordinator.GetField("field1")
    	if field.WouldChange(StateSuccess, nil) {
        	field.SetSuccess()
    	}
*/

package hydration

import (
	"errors"
	"fmt"
)

type Coordinator struct {
	fieldMap               map[string]*Field
	mandatorySuccessFields []string
}

type HydrationSummary struct {
	AllFinished        bool
	AllSuccess         bool
	MandatorySatisfied bool
	AnyError           bool
}

var ErrFieldAlreadyExist = errors.New("field already exist")
var ErrFieldNotRegistered = errors.New("field not registered")

func NewCoordinator() *Coordinator {
	return &Coordinator{
		fieldMap: map[string]*Field{},
	}
}

func (c *Coordinator) Register(fieldName string, errorRarity ErrorRarity, errorRetryable bool, successMandatory bool) error {
	if _, exists := c.fieldMap[fieldName]; exists {
		return fmt.Errorf("field %q error: %w", fieldName, ErrFieldAlreadyExist)
	}
	c.fieldMap[fieldName] = &Field{
		state:            StateNotAsked,
		errorRarity:      errorRarity,
		errorRetryable:   errorRetryable,
		successMandatory: successMandatory,
	}
	if successMandatory {
		c.mandatorySuccessFields = append(c.mandatorySuccessFields, fieldName)
	}
	return nil
}

func (c *Coordinator) GetField(fieldName string) (*Field, error) {
	field, ok := c.fieldMap[fieldName]
	if !ok {
		return nil, fmt.Errorf("field %s error: %w", fieldName, ErrFieldNotRegistered)
	}
	return field, nil
}

func (c *Coordinator) ResetField(fieldName string) {
	field := c.fieldMap[fieldName]
	field.SetNotAsked()
}

func (c *Coordinator) HydrateField(fieldName string) {
	field := c.fieldMap[fieldName]
	field.SetHydrating()
}

func (c *Coordinator) ResetAllFields() {
	for name := range c.fieldMap {
		c.ResetField(name)
	}
}

func (c *Coordinator) AllFinished() bool {
	for _, field := range c.fieldMap {
		if !field.IsFinished() {
			return false
		}
	}
	return true
}

func (c *Coordinator) AllSuccess() bool {
	if !c.AllFinished() {
		return false
	}
	for _, field := range c.fieldMap {
		if !field.IsSuccess() {
			return false
		}
	}
	return true
}

func (c *Coordinator) AllHydrating() bool {
	for _, field := range c.fieldMap {
		if !field.IsHydrating() {
			return false
		}
	}
	return true
}

func (c *Coordinator) AnyHydrating() bool {
	for _, field := range c.fieldMap {
		if field.IsHydrating() {
			return true
		}
	}
	return false
}

func (c *Coordinator) ErrorsExist() bool {
	for _, field := range c.fieldMap {
		if field.IsError() {
			return true
		}
	}
	return false
}

func (c *Coordinator) MandatorySucceeded() bool {
	mandatorySuccessCount := 0
	for _, fieldName := range c.mandatorySuccessFields {
		if c.fieldMap[fieldName].CanProcess() {
			mandatorySuccessCount++
		}
	}
	return mandatorySuccessCount == len(c.mandatorySuccessFields)
}

func (c *Coordinator) GetFieldsForRetry() []string {
	fieldsToRetry := []string{}

	// Maps don't guarantee order, but it's okay since the retry execution will be parallel
	for name, field := range c.fieldMap {
		if field.IsError() {
			fieldsToRetry = append(fieldsToRetry, name)
		}
	}

	return fieldsToRetry
}

func (c *Coordinator) GetErrors() map[string]error {
	errors := make(map[string]error)
	for name, field := range c.fieldMap {
		if field.IsError() {
			errors[name] = field.err
		}
	}
	return errors
}

func (c *Coordinator) GetErrorsAsString() []string {
	if !c.ErrorsExist() {
		return []string{}
	}

	errorStrings := []string{}
	for name, field := range c.fieldMap {
		if field.err != nil {
			retryable := "permanent"
			if field.errorRetryable {
				retryable = "retryable"
			}
			prefix := fmt.Sprintf("[%s][%s][%s]", field.errorRarity, retryable, name)
			errorStrings = append(errorStrings, prefix+" "+field.err.Error())
		}
	}
	return errorStrings
}

func (c *Coordinator) HydrateAllFields() {
	for _, field := range c.fieldMap {
		field.SetHydrating()
	}
}

func (c *Coordinator) ComputeHydrationSummary() HydrationSummary {
	return HydrationSummary{
		AllFinished:        c.AllFinished(),
		AllSuccess:         c.AllSuccess(),
		MandatorySatisfied: c.MandatorySucceeded(),
		AnyError:           c.ErrorsExist(),
	}
}

func (c *Coordinator) GetHydrationProgress() (completed, total int) {
	total = len(c.fieldMap)
	for _, field := range c.fieldMap {
		if field.IsFinished() {
			completed++
		}
	}
	return completed, total
}
