package network

import (
	"github.com/khoahotran/gochain-ledger/domain"
	"github.com/khoahotran/gochain-ledger/proto"
)

func MapDomainTransactionToProto(tx *domain.Transaction) *proto.Transaction {
	vin := make([]*proto.TxInput, len(tx.Vin))
	for i, in := range tx.Vin {
		vin[i] = &proto.TxInput{
			TxId:      in.TxID,
			VoutIndex: int32(in.VoutIndex),
			Signature: in.Signature,
			PublicKey: in.PublicKey,
		}
	}

	vout := make([]*proto.TxOutput, len(tx.Vout))
	for i, out := range tx.Vout {
		vout[i] = &proto.TxOutput{
			Value:      out.Value,
			PubKeyHash: out.PubKeyHash,
		}
	}

	return &proto.Transaction{
		Id:      tx.ID,
		Vin:     vin,
		Vout:    vout,
		Type:    int32(tx.Type),
		Payload: tx.Payload,
	}
}

func MapDomainBlockToProto(b *domain.Block) *proto.Block {
	txs := make([]*proto.Transaction, len(b.Transactions))
	for i, tx := range b.Transactions {
		txs[i] = MapDomainTransactionToProto(tx)
	}

	return &proto.Block{
		Timestamp:     b.Timestamp,
		PrevBlockHash: b.PrevBlockHash,
		Hash:          b.Hash,
		Transactions:  txs,
		Nonce:         b.Nonce,
	}
}

func MapProtoTransactionToDomain(tx *proto.Transaction) *domain.Transaction {
	vin := make([]domain.TxInput, len(tx.Vin))
	for i, in := range tx.Vin {
		vin[i] = domain.TxInput{
			TxID:      in.TxId,
			VoutIndex: int(in.VoutIndex),
			Signature: in.Signature,
			PublicKey: in.PublicKey,
		}
	}

	vout := make([]domain.TxOutput, len(tx.Vout))
	for i, out := range tx.Vout {
		vout[i] = domain.TxOutput{
			Value:      out.Value,
			PubKeyHash: out.PubKeyHash,
		}
	}

	return &domain.Transaction{
		ID:      tx.Id,
		Vin:     vin,
		Vout:    vout,
		Type:    domain.TxType(tx.Type),
		Payload: tx.Payload,
	}
}

func MapProtoBlockToDomain(b *proto.Block) *domain.Block {
	txs := make([]*domain.Transaction, len(b.Transactions))
	for i, tx := range b.Transactions {
		txs[i] = MapProtoTransactionToDomain(tx)
	}

	return &domain.Block{
		Timestamp:     b.Timestamp,
		PrevBlockHash: b.PrevBlockHash,
		Hash:          b.Hash,
		Transactions:  txs,
		Nonce:         b.Nonce,
	}
}
