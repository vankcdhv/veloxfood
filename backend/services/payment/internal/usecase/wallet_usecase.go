package usecase

import (
	"context"

	"project/services/payment/internal/entity"
	"project/services/payment/internal/repository"
)

// WalletUsecase handles customer wallet queries.
type WalletUsecase interface {
	// GetWalletWithHistory returns the customer's wallet and recent ledger entries.
	GetWalletWithHistory(ctx context.Context, customerID string, limit int) (*entity.Wallet, []*entity.LedgerEntry, error)
}

type walletUsecase struct {
	walletRepo repository.WalletRepository
	ledgerRepo repository.LedgerRepository
}

func NewWalletUsecase(walletRepo repository.WalletRepository, ledgerRepo repository.LedgerRepository) WalletUsecase {
	return &walletUsecase{walletRepo: walletRepo, ledgerRepo: ledgerRepo}
}

func (uc *walletUsecase) GetWalletWithHistory(ctx context.Context, customerID string, limit int) (*entity.Wallet, []*entity.LedgerEntry, error) {
	wallet, err := uc.walletRepo.GetByOwner(ctx, entity.WalletOwnerCustomer, customerID)
	if err != nil {
		return nil, nil, err
	}
	// Return zero-balance wallet stub if never created (customer hasn't transacted yet).
	if wallet == nil {
		wallet = &entity.Wallet{
			OwnerType: entity.WalletOwnerCustomer,
			OwnerID:   customerID,
			Balance:   0,
		}
		return wallet, nil, nil
	}

	entries, err := uc.ledgerRepo.ListByWallet(ctx, wallet.ID, limit, 0)
	if err != nil {
		return nil, nil, err
	}
	return wallet, entries, nil
}
