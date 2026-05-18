package db

import (
	"database/sql"
	"fmt"
	"time"

	"crosschain/internal/swap"

	_ "github.com/lib/pq"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func (s *PostgresStore) CreateSwap(sw *swap.Swap) error {
	const query = `
		INSERT INTO swaps (
			id, from_chain, to_chain, from_asset, to_asset, amount_in, amount_out, fee,
			deposit_address, dest_address, status, btc_amount, trade1_order_id,
			trade2_order_id, withdrawal_tx_id, reference
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NULLIF($12, ''),
			NULLIF($13, ''), NULLIF($14, ''), NULLIF($15, ''), $16)
	`

	_, err := s.db.Exec(query,
		sw.ID,
		sw.FromChain,
		sw.ToChain,
		sw.FromAsset,
		sw.ToAsset,
		sw.AmountIn,
		sw.AmountOut,
		sw.Fee,
		sw.DepositAddress,
		sw.DestAddress,
		sw.Status,
		sw.BTCAmount,
		sw.Trade1OrderID,
		sw.Trade2OrderID,
		sw.WithdrawalTxID,
		sw.Reference,
	)
	if err != nil {
		return fmt.Errorf("insert swap: %w", err)
	}

	return nil
}

func (s *PostgresStore) GetSwapByID(id string) (*swap.Swap, error) {
	return s.getSwap("id = $1", id)
}

func (s *PostgresStore) GetSwapByDepositAddress(addr string) (*swap.Swap, error) {
	return s.getSwap("deposit_address = $1", addr)
}

func (s *PostgresStore) GetSwapByReference(ref string) (*swap.Swap, error) {
	return s.getSwap("reference = $1", ref)
}

func (s *PostgresStore) UpdateSwapStatus(id string, status swap.Status) error {
	const query = `UPDATE swaps SET status = $2, updated_at = NOW() WHERE id = $1`
	return s.execOne(query, id, status)
}

func (s *PostgresStore) UpdateSwapAfterTrades(id, btcAmount, trade1ID, trade2ID string) error {
	const query = `
		UPDATE swaps
		SET btc_amount = NULLIF($2, ''),
			trade1_order_id = NULLIF($3, ''),
			trade2_order_id = NULLIF($4, ''),
			updated_at = NOW()
		WHERE id = $1
	`
	return s.execOne(query, id, btcAmount, trade1ID, trade2ID)
}

func (s *PostgresStore) SetWithdrawalTxID(id, txID string) error {
	const query = `
		UPDATE swaps
		SET withdrawal_tx_id = $2, updated_at = NOW()
		WHERE id = $1
	`
	return s.execOne(query, id, txID)
}

func (s *PostgresStore) getSwap(where string, args ...any) (*swap.Swap, error) {
	query := `
		SELECT id, from_chain, to_chain, from_asset, to_asset, amount_in, amount_out, fee,
			deposit_address, dest_address, status, btc_amount, trade1_order_id,
			trade2_order_id, withdrawal_tx_id, reference, created_at, updated_at
		FROM swaps
		WHERE ` + where + `
		LIMIT 1
	`

	row := s.db.QueryRow(query, args...)
	sw, err := scanSwap(row)
	if err != nil {
		return nil, err
	}

	return sw, nil
}

func (s *PostgresStore) execOne(query string, args ...any) error {
	result, err := s.db.Exec(query, args...)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanSwap(row scanner) (*swap.Swap, error) {
	var sw swap.Swap
	var status string
	var btcAmount sql.NullString
	var trade1OrderID sql.NullString
	var trade2OrderID sql.NullString
	var withdrawalTxID sql.NullString
	var createdAt time.Time
	var updatedAt time.Time

	err := row.Scan(
		&sw.ID,
		&sw.FromChain,
		&sw.ToChain,
		&sw.FromAsset,
		&sw.ToAsset,
		&sw.AmountIn,
		&sw.AmountOut,
		&sw.Fee,
		&sw.DepositAddress,
		&sw.DestAddress,
		&status,
		&btcAmount,
		&trade1OrderID,
		&trade2OrderID,
		&withdrawalTxID,
		&sw.Reference,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	sw.Status = swap.Status(status)
	sw.BTCAmount = btcAmount.String
	sw.Trade1OrderID = trade1OrderID.String
	sw.Trade2OrderID = trade2OrderID.String
	sw.WithdrawalTxID = withdrawalTxID.String
	sw.CreatedAt = createdAt
	sw.UpdatedAt = updatedAt

	return &sw, nil
}
