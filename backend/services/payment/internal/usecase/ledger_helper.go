package usecase

import (
	"context"

	"project/services/payment/internal/entity"
	"project/services/payment/internal/repository"

	"gorm.io/gorm"
)

// transferParams holds everything needed for a double-entry money transfer.
type transferParams struct {
	tx         *gorm.DB
	walletRepo repository.WalletRepository
	ledgerRepo repository.LedgerRepository

	debitOwnerType  entity.WalletOwnerType
	debitOwnerID    string
	creditOwnerType entity.WalletOwnerType
	creditOwnerID   string

	entryType entity.LedgerEntryType
	refType   entity.LedgerRefType
	refID     string
	amount    int64 // positive VND amount to move
	traceID   string
}

// doubleEntryTransfer executes one complete double-entry transfer:
//   - locks debit wallet FOR UPDATE, checks balance >= amount
//   - locks credit wallet FOR UPDATE
//   - writes debit ledger entry (−amount) and credit ledger entry (+amount)
//   - updates both wallet balances
//
// All operations run within the caller-supplied tx.
// Invariant: sum of the two ledger entry amounts == 0.
func doubleEntryTransfer(ctx context.Context, p transferParams) error {
	// Lock debit wallet and check balance.
	debitWallet, err := p.walletRepo.GetOrCreateForUpdate(ctx, p.tx, p.debitOwnerType, p.debitOwnerID)
	if err != nil {
		return err
	}
	// The SYSTEM wallet is an internal clearing/float account: a customer top-up is
	// external money (from MoMo) flowing in, so debiting SYSTEM may push it negative
	// — that negative balance is the liability owed back to customers, which is
	// correct. Only customer/store wallets must never overdraw.
	if p.debitOwnerType != entity.WalletOwnerSystem && debitWallet.Balance < p.amount {
		return ErrInsufficientBalance
	}

	// Lock credit wallet.
	creditWallet, err := p.walletRepo.GetOrCreateForUpdate(ctx, p.tx, p.creditOwnerType, p.creditOwnerID)
	if err != nil {
		return err
	}

	newDebitBalance := debitWallet.Balance - p.amount
	newCreditBalance := creditWallet.Balance + p.amount

	tracePtr := &p.traceID
	if p.traceID == "" {
		tracePtr = nil
	}

	// Debit entry (negative amount).
	if err := p.ledgerRepo.Append(ctx, p.tx, &entity.LedgerEntry{
		WalletID:     debitWallet.ID,
		EntryType:    p.entryType,
		Amount:       -p.amount,
		RefType:      p.refType,
		RefID:        p.refID,
		BalanceAfter: newDebitBalance,
		TraceID:      tracePtr,
	}); err != nil {
		return err
	}

	// Credit entry (positive amount).
	if err := p.ledgerRepo.Append(ctx, p.tx, &entity.LedgerEntry{
		WalletID:     creditWallet.ID,
		EntryType:    p.entryType,
		Amount:       p.amount,
		RefType:      p.refType,
		RefID:        p.refID,
		BalanceAfter: newCreditBalance,
		TraceID:      tracePtr,
	}); err != nil {
		return err
	}

	// Update balances.
	if err := p.walletRepo.UpdateBalance(ctx, p.tx, debitWallet.ID, newDebitBalance); err != nil {
		return err
	}
	return p.walletRepo.UpdateBalance(ctx, p.tx, creditWallet.ID, newCreditBalance)
}
