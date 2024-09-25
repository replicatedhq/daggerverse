package main

import (
	"context"
	"dagger/onepassword/internal/dagger"
	"errors"
	"fmt"

	onepassword "github.com/1password/onepassword-sdk-go"
)

type Onepassword struct{}

var (
	ErrVaultNotFound   = errors.New("vault not found")
	ErrItemNotFound    = errors.New("item not found")
	ErrFieldNotFound   = errors.New("field not found")
	ErrSectionNotFound = errors.New("section not found")
)

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
