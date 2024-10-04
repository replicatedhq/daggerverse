package main

import (
	"context"
	"dagger/onepassword/internal/dagger"
	"encoding/json"
	"errors"
	"fmt"

	onepassword "github.com/1password/onepassword-sdk-go"
)

type Onepassword struct{}

var (
	ErrVaultNotFound        = errors.New("vault not found")
	ErrItemNotFound         = errors.New("item not found")
	ErrFieldNotFound        = errors.New("field not found")
	ErrSectionNotFound      = errors.New("section not found")
	ErrRotationSpecNotFound = errors.New("rotation specs not found")
)

// SecretRotationSpecs is a struct that contains the information needed to rotate a key
//
//	this is how it is stored in the OnePassword vault item
type SecretRotationSpecs struct {
	// Date types are not supported in the onepassword SDK. They get returned as null and type 'unsupported'
	// So we store them as strings and will have to convert them to date types in the code
	// for this reason, we need to set a standard date format for the string when entering in onepassword
	ExpiresOn        string // The date the key expires as a string
	CreatedOn        string // The date the key was created as a string
	RotationFunction string // The function to rotate the key
}

// Returns the value of a secret from the specificed vault, with the specified name and field.
func (m *Onepassword) FindSecret(
	ctx context.Context,

	// 1password service account
	serviceAccount *dagger.Secret,

	// Name of the vault to search
	vaultName string,

	// Name of the item to find
	itemName string,

	// Name of the field to find
	fieldName string,

	// Limit to a specific section of the item
	// +optional
	section string,
) (*dagger.Secret, error) {
	serviceAccountPlaintext, err := serviceAccount.Plaintext(ctx)
	if err != nil {
		panic(err)
	}

	client, err := onepassword.NewClient(ctx,
		onepassword.WithServiceAccountToken(serviceAccountPlaintext),
		onepassword.WithIntegrationInfo("Dagger Workflow", "v0.0.1"),
	)
	if err != nil {
		panic(err)
	}

	vault, err := findVault(ctx, client, vaultName)
	if err != nil {
		return nil, err
	}

	itemOverview, err := findItem(ctx, client, vault.ID, itemName)
	if err != nil {
		return nil, err
	}

	item, err := client.Items.Get(ctx, vault.ID, itemOverview.ID)

	sectionID, err := findSectionID(item, section)
	if err != nil {
		return nil, err
	}

	for _, field := range item.Fields {
		if section == "" || (field.SectionID != nil && *field.SectionID == sectionID) {
			if field.Title == fieldName {
				return dagger.Connect().SetSecret(fieldName, field.Value), nil
			}
		}
	}

	return nil, ErrFieldNotFound
}

// Returns the specifications to rotate the secrets in the specified vault.
func (m *Onepassword) FindSecretRotationSpecs(
	ctx context.Context,

	// 1password service account
	serviceAccount *dagger.Secret,

	// Name of the vault to search
	vaultName string,

	// Name of the item to find
	itemName string,

	// Section name where rotation specs are stored
	sectionName string,
) (*dagger.Secret, error) {
	serviceAccountPlaintext, err := serviceAccount.Plaintext(ctx)
	if err != nil {
		panic(err)
	}
	client, err := onepassword.NewClient(ctx,
		onepassword.WithServiceAccountToken(serviceAccountPlaintext),
		onepassword.WithIntegrationInfo("Dagger Workflow", "v0.0.1"),
	)
	if err != nil {
		panic(err)
	}

	// Find the vault
	vault, err := findVault(ctx, client, vaultName)
	if err != nil {
		return nil, err
	}

	// Find the item
	itemOverview, err := findItem(ctx, client, vault.ID, itemName)
	if err != nil {
		return nil, err
	}

	item, err := client.Items.Get(ctx, vault.ID, itemOverview.ID)

	// Find the section where the rotation specs are stored
	// reccomend the section be named "rotation"
	sectionID, err := findSectionID(item, sectionName)
	if err != nil {
		return nil, err
	}

	rotationInfo := SecretRotationSpecs{}
	// Iterate through the fields in the section and store the values in a struct

	for _, field := range item.Fields {
		if *field.SectionID == sectionID {
			switch field.Title {
			case "expires-on":
				rotationInfo.ExpiresOn = field.Value
			case "created-on":
				rotationInfo.CreatedOn = field.Value
			case "rotationFunction":
				rotationInfo.RotationFunction = field.Value
			}
		}
		// If all values are found, break the loop
		if rotationInfo.ExpiresOn != "" && rotationInfo.CreatedOn != "" && rotationInfo.RotationFunction != "" {
			break
		}
	}

	// If we don't have all values, return an error
	if rotationInfo.ExpiresOn == "" || rotationInfo.CreatedOn == "" || rotationInfo.RotationFunction == "" {
		return nil, ErrRotationSpecNotFound
	}

	// Convert the struct to JSON format (or any other format you prefer)
	rotationSpecsJSON, err := json.Marshal(rotationInfo)
	if err != nil {
		return nil, err
	}

	// Return the combined rotation specs as a secret in JSON format
	return dagger.Connect().SetSecret("rotationSpecs", string(rotationSpecsJSON)), nil

}

// Set the value of a secret in the specified vault, with the specified name and field.
func (m *Onepassword) PutSecret(
	ctx context.Context,

	// 1password service account
	serviceAccount *dagger.Secret,

	// Name of the vault to search
	vaultName string,

	// Name of the item to find
	itemName string,

	// Name of the field to find
	fieldName string,

	// Value to set
	value string,
) error {
	serviceAccountPlaintext, err := serviceAccount.Plaintext(ctx)
	if err != nil {
		panic(err)
	}

	client, err := onepassword.NewClient(ctx,
		onepassword.WithServiceAccountToken(serviceAccountPlaintext),
		onepassword.WithIntegrationInfo("Dagger Workflow", "v0.0.1"),
	)
	if err != nil {
		panic(err)
	}

	vault, err := findVault(ctx, client, vaultName)
	if err != nil {
		return err
	}

	var itemOverview *onepassword.ItemOverview
	io, err := findItem(ctx, client, vault.ID, itemName)
	if err != nil {
		if err == ErrItemNotFound {
			_, err = client.Items.Create(ctx, onepassword.ItemCreateParams{
				Title: itemName,
			})
			if err != nil {
				return err
			}
			itemOverview = io
		} else {
			return err
		}
	} else {
		itemOverview = io
	}

	fmt.Printf("itemOverview: %+v\n", itemOverview)

	return nil
}

func findVault(ctx context.Context, client *onepassword.Client, vaultName string) (*onepassword.VaultOverview, error) {
	vaultsIterator, err := client.Vaults.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	for {
		item, err := vaultsIterator.Next()
		if err != nil {
			if err == onepassword.ErrorIteratorDone {
				return nil, ErrVaultNotFound
			}

			return nil, err
		}

		if item.Title == vaultName {
			return item, nil
		}
	}
}

func findItem(ctx context.Context, client *onepassword.Client, vaultID string, itemName string) (*onepassword.ItemOverview, error) {
	itemsIterator, err := client.Items.ListAll(ctx, vaultID)
	if err != nil {
		return nil, err
	}

	for {
		i, err := itemsIterator.Next()
		if err != nil {
			if err == onepassword.ErrorIteratorDone {
				return nil, ErrItemNotFound
			}

			return nil, err
		}

		if i.Title == itemName {
			return i, nil
		}
	}
}

// findSectionID finds the ID of a section name. The name is case sensitive
func findSectionID(item onepassword.Item, sectionName string) (string, error) {
	if sectionName == "" {
		return "", nil
	}

	for _, s := range item.Sections {
		if s.Title == sectionName {
			return s.ID, nil
		}
	}
	return "", ErrSectionNotFound
}
