package api

import (
	"net/http"

	"github.com/NebulousLabs/Sia/build"
	"github.com/NebulousLabs/Sia/types"
)

type (
	// ExplorerBlock is a block with some extra information such as the id and
	// height. This information is provided for programs that may not be
	// complex enough to compute the ID on their own.
	ExplorerBlock struct {
		ID             types.BlockID           `json:"id"`
		Height         types.BlockHeight       `json:"height"`
		Transactions   []ExplorerTransaction   `json:"transactions"`
		MinerPayoutIDs []types.SiacoinOutputID `json:"minerpayoutids"`
		RawBlock       types.Block             `json:"rawblock"`
	}

	// ExplorerTransaction is a transcation with some extra information such as
	// the parent block. This information is provided for programs that may not
	// be complex enough to compute the extra information on their own.
	ExplorerTransaction struct {
		ID                                       types.TransactionID       `json:"id"`
		Height                                   types.BlockHeight         `json:"height"`
		Parent                                   types.BlockID             `json:"parent"`
		SiacoinOutputIDs                         []types.SiacoinOutputID   `json:"siacoinoutputids"`
		FileContractIDs                          []types.FileContractID    `json:"filecontractids"`
		FileContractValidProofOutputIDs          [][]types.SiacoinOutputID `json:"filecontractvalidproofoutputids"`          // outer array is per-contract
		FileContractMissedProofOutputIDs         [][]types.SiacoinOutputID `json:"filecontractmissedproofoutputids"`         // outer array is per-contract
		FileContractRevisionValidProofOutputIDs  [][]types.SiacoinOutputID `json:"filecontractrevisionvalidproofoutputids"`  // outer array is per-revision
		FileContractRevisionMissedProofOutputIDs [][]types.SiacoinOutputID `json:"filecontractrevisionmissedproofoutputids"` // outer array is per-revision
		SiafundOutputIDs                         []types.SiafundOutputID   `json:"siafundoutputids"`
		SiaClaimOutputIDs                        []types.SiacoinOutputID   `json:"siafundclaimoutputids"`
		RawTransaction                           types.Transaction         `json:"rawtransaction"`
	}

	// ExplorerGET is the object returned as a response to a GET request to
	// /explorer.
	ExplorerGET struct {
		// General consensus information.
		Height            types.BlockHeight `json:"height"`
		CurrentBlock      types.BlockID     `json:"currentblock"`
		Target            types.Target      `json:"target"`
		Difficulty        types.Currency    `json:"difficulty"`
		MaturityTimestamp types.Timestamp   `json:"maturitytimestamp"`
		TotalCoins        types.Currency    `json:"totalcoins"`

		// Information about transaction type usage.
		TransactionCount          uint64 `json:"transactioncount"`
		SiacoinInputCount         uint64 `json:"siacoininputcount"`
		SiacoinOutputCount        uint64 `json:"siacoinoutputcount"`
		FileContractCount         uint64 `json:"filecontractcount"`
		FileContractRevisionCount uint64 `json:"filecontractrevisioncount"`
		StorageProofCount         uint64 `json:"storageproofcount"`
		SiafundInputCount         uint64 `json:"siafundinputcount"`
		SiafundOutputCount        uint64 `json:"siafundoutputcount"`
		MinerFeeCount             uint64 `json:"minerfeecount"`
		ArbitraryDataCount        uint64 `json:"arbitrarydatacount"`
		TransactionSignatureCount uint64 `json:"transactionsignaturecount"`

		// Information about file contracts and file contract revisions.
		ActiveContractCount uint64         `json:"activecontractcount"`
		ActiveContractCost  types.Currency `json:"activecontractcost"`
		ActiveContractSize  types.Currency `json:"activecontractsize"`
		TotalContractCost   types.Currency `json:"totalcontractcost"`
		TotalContractSize   types.Currency `json:"totalcontractsize"`
	}

	// ExplorerHashGET is the object returned as a response to a GET request to
	// /explorer/hash. The HashType will indicate whether the hash corresponds
	// to a block id, a transaction id, a siacoin output id, a file contract
	// id, or a siafund output id. In the case of a block id, 'Block' will be
	// filled out and all the rest of the fields will be blank. In the case of
	// a transaction id, 'Transaction' will be filled out and all the rest of
	// the fields will be blank. For everything else, 'Transactions' and
	// 'Blocks' will/may be filled out and everything else will be blank.
	ExplorerHashGET struct {
		HashType     string                `json:"hashtype"`
		Block        ExplorerBlock         `json:"block"`
		Blocks       []types.BlockID       `json:"blocks"`
		Transaction  ExplorerTransaction   `json:"transaction"`
		Transactions []types.TransactionID `json:"transactions"`
	}
)

// explorerHandlerGET handles GET requests to /explorer.
func (srv *Server) explorerHandlerGET(w http.ResponseWriter, req *http.Request) {
	stats := srv.explorer.Statistics()
	writeJSON(w, ExplorerGET{
		Height:            stats.Height,
		CurrentBlock:      stats.CurrentBlock,
		Target:            stats.Target,
		Difficulty:        stats.Difficulty,
		MaturityTimestamp: stats.MaturityTimestamp,
		TotalCoins:        stats.TotalCoins,

		TransactionCount:          stats.TransactionCount,
		SiacoinInputCount:         stats.SiacoinInputCount,
		SiacoinOutputCount:        stats.SiacoinOutputCount,
		FileContractCount:         stats.FileContractCount,
		FileContractRevisionCount: stats.FileContractRevisionCount,
		StorageProofCount:         stats.StorageProofCount,
		SiafundInputCount:         stats.SiafundInputCount,
		SiafundOutputCount:        stats.SiafundOutputCount,
		MinerFeeCount:             stats.MinerFeeCount,
		ArbitraryDataCount:        stats.ArbitraryDataCount,
		TransactionSignatureCount: stats.TransactionSignatureCount,

		ActiveContractCount: stats.ActiveContractCount,
		ActiveContractCost:  stats.ActiveContractCost,
		ActiveContractSize:  stats.ActiveContractSize,
		TotalContractCost:   stats.TotalContractCost,
		TotalContractSize:   stats.TotalContractSize,
	})
}

// explorerHandler handles API calls to /explorer.
func (srv *Server) explorerHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method == "" || req.Method == "GET" {
		srv.explorerHandlerGET(w, req)
		return
	}
	writeError(w, "unrecognized method when calling /explorer", http.StatusBadRequest)
}

// explorerHashHandlerGET handles GET requests to /explorer/hash.
func (srv *Server) explorerHashHandlerGET(w http.ResponseWriter, req *http.Request) {
	// The hash is scanned as an address, because an address can be typecast to
	// all other necessary types, and will correclty decode hashes whether or
	// not they have a checksum.
	hash, err := scanAddress(req.FormValue("hash"))
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Try the hash as a block id.
	block, height, exists := srv.explorer.Block(types.BlockID(hash))
	if exists {
		var mpoids []types.SiacoinOutputID
		var etxns []ExplorerTransaction
		for i := range block.MinerPayouts {
			mpoids = append(mpoids, block.MinerPayoutID(uint64(i)))
		}
		for _, txn := range block.Transactions {
			var scoids []types.SiacoinOutputID
			var fcids []types.FileContractID
			var fcvpoidss [][]types.SiacoinOutputID
			var fcmpoidss [][]types.SiacoinOutputID
			var fcrvpoidss [][]types.SiacoinOutputID
			var fcrmpoidss [][]types.SiacoinOutputID
			var sfoids []types.SiafundOutputID
			var sfcoids []types.SiacoinOutputID
			for i := range txn.SiacoinOutputs {
				scoids = append(scoids, txn.SiacoinOutputID(uint64(i)))
			}
			for i, fc := range txn.FileContracts {
				fcid := txn.FileContractID(uint64(i))
				var fcvpoids []types.SiacoinOutputID
				var fcmpoids []types.SiacoinOutputID
				for j := range fc.ValidProofOutputs {
					fcvpoids = append(fcvpoids, fcid.StorageProofOutputID(types.ProofValid, uint64(j)))
				}
				for j := range fc.MissedProofOutputs {
					fcmpoids = append(fcmpoids, fcid.StorageProofOutputID(types.ProofMissed, uint64(j)))
				}
				fcids = append(fcids, fcid)
				fcvpoidss = append(fcvpoidss, fcvpoids)
				fcmpoidss = append(fcmpoidss, fcmpoids)
			}
			for _, fcr := range txn.FileContractRevisions {
				var fcrvpoids []types.SiacoinOutputID
				var fcrmpoids []types.SiacoinOutputID
				for j := range fcr.NewValidProofOutputs {
					fcrvpoids = append(fcrvpoids, fcr.ParentID.StorageProofOutputID(types.ProofValid, uint64(j)))
				}
				for j := range fcr.NewMissedProofOutputs {
					fcrmpoids = append(fcrmpoids, fcr.ParentID.StorageProofOutputID(types.ProofMissed, uint64(j)))
				}
				fcrvpoidss = append(fcrvpoidss, fcrvpoids)
				fcrmpoidss = append(fcrmpoidss, fcrmpoids)
			}
			for i := range txn.SiafundOutputs {
				sfoids = append(sfoids, txn.SiafundOutputID(uint64(i)))
			}
			for _, sfi := range txn.SiafundInputs {
				sfcoids = append(sfcoids, sfi.ParentID.SiaClaimOutputID())
			}
			etxns = append(etxns, ExplorerTransaction{
				ID:                               txn.ID(),
				Height:                           height,
				Parent:                           block.ID(),
				SiacoinOutputIDs:                 scoids,
				FileContractIDs:                  fcids,
				FileContractValidProofOutputIDs:  fcvpoidss,
				FileContractMissedProofOutputIDs: fcmpoidss,
				SiafundOutputIDs:                 sfoids,
				SiaClaimOutputIDs:                sfcoids,
				RawTransaction:                   txn,
			})
		}
		writeJSON(w, ExplorerHashGET{
			HashType: "blockid",
			Block: ExplorerBlock{
				ID:             block.ID(),
				Height:         height,
				Transactions:   etxns,
				MinerPayoutIDs: mpoids,
				RawBlock:       block,
			},
		})
		return
	}

	// Try the hash as a transaction id.
	block, height, exists = srv.explorer.Transaction(types.TransactionID(hash))
	if exists {
		var txn types.Transaction
		for _, t := range block.Transactions {
			if t.ID() == types.TransactionID(hash) {
				txn = t
			}
		}
		var scoids []types.SiacoinOutputID
		var fcids []types.FileContractID
		var fcvpoidss [][]types.SiacoinOutputID
		var fcmpoidss [][]types.SiacoinOutputID
		var fcrvpoidss [][]types.SiacoinOutputID
		var fcrmpoidss [][]types.SiacoinOutputID
		var sfoids []types.SiafundOutputID
		var sfcoids []types.SiacoinOutputID
		for i := range txn.SiacoinOutputs {
			scoids = append(scoids, txn.SiacoinOutputID(uint64(i)))
		}
		for i, fc := range txn.FileContracts {
			fcid := txn.FileContractID(uint64(i))
			var fcvpoids []types.SiacoinOutputID
			var fcmpoids []types.SiacoinOutputID
			for j := range fc.ValidProofOutputs {
				fcvpoids = append(fcvpoids, fcid.StorageProofOutputID(types.ProofValid, uint64(j)))
			}
			for j := range fc.MissedProofOutputs {
				fcmpoids = append(fcmpoids, fcid.StorageProofOutputID(types.ProofMissed, uint64(j)))
			}
			fcids = append(fcids, fcid)
			fcvpoidss = append(fcvpoidss, fcvpoids)
			fcmpoidss = append(fcmpoidss, fcmpoids)
		}
		for _, fcr := range txn.FileContractRevisions {
			var fcrvpoids []types.SiacoinOutputID
			var fcrmpoids []types.SiacoinOutputID
			for j := range fcr.NewValidProofOutputs {
				fcrvpoids = append(fcrvpoids, fcr.ParentID.StorageProofOutputID(types.ProofValid, uint64(j)))
			}
			for j := range fcr.NewMissedProofOutputs {
				fcrmpoids = append(fcrmpoids, fcr.ParentID.StorageProofOutputID(types.ProofMissed, uint64(j)))
			}
			fcrvpoidss = append(fcrvpoidss, fcrvpoids)
			fcrmpoidss = append(fcrmpoidss, fcrmpoids)
		}
		for i := range txn.SiafundOutputs {
			sfoids = append(sfoids, txn.SiafundOutputID(uint64(i)))
		}
		for _, sfi := range txn.SiafundInputs {
			sfcoids = append(sfcoids, sfi.ParentID.SiaClaimOutputID())
		}
		writeJSON(w, ExplorerHashGET{
			HashType: "transactionid",
			Transaction: ExplorerTransaction{
				ID:                               txn.ID(),
				Height:                           height,
				Parent:                           block.ID(),
				SiacoinOutputIDs:                 scoids,
				FileContractIDs:                  fcids,
				FileContractValidProofOutputIDs:  fcvpoidss,
				FileContractMissedProofOutputIDs: fcmpoidss,
				SiafundOutputIDs:                 sfoids,
				SiaClaimOutputIDs:                sfcoids,
				RawTransaction:                   txn,
			},
		})
		return
	}

	// Try the hash as an unlock hash.
	txids := srv.explorer.UnlockHash(types.UnlockHash(hash))
	if len(txids) != 0 {
		var txns []types.TransactionID
		var blocks []types.BlockID
		for _, txid := range txids {
			block, _, exists := srv.explorer.Transaction(txid)
			if !exists && build.DEBUG {
				panic("explorer pointing to nonexistant txn")
			}
			if types.TransactionID(block.ID()) == txid {
				blocks = append(blocks, block.ID())
			} else {
				txns = append(txns, txid)
			}
		}
		writeJSON(w, ExplorerHashGET{
			HashType:     "unlockhash",
			Blocks:       blocks,
			Transactions: txns,
		})
		return
	}

	// Try the hash as a siacoin output id.
	txids = srv.explorer.SiacoinOutputID(types.SiacoinOutputID(hash))
	if len(txids) != 0 {
		var txns []types.TransactionID
		var blocks []types.BlockID
		for _, txid := range txids {
			block, _, exists := srv.explorer.Transaction(txid)
			if !exists && build.DEBUG {
				panic("explorer pointing to nonexistant txn")
			}
			if types.TransactionID(block.ID()) == txid {
				blocks = append(blocks, block.ID())
			} else {
				txns = append(txns, txid)
			}
		}
		writeJSON(w, ExplorerHashGET{
			HashType:     "siacoinoutputid",
			Blocks:       blocks,
			Transactions: txns,
		})
		return
	}

	// Try the hash as a file contract id.
	txids = srv.explorer.FileContractID(types.FileContractID(hash))
	if len(txids) != 0 {
		var txns []types.TransactionID
		var blocks []types.BlockID
		for _, txid := range txids {
			block, _, exists := srv.explorer.Transaction(txid)
			if !exists && build.DEBUG {
				panic("explorer pointing to nonexistant txn")
			}
			if types.TransactionID(block.ID()) == txid {
				blocks = append(blocks, block.ID())
			} else {
				txns = append(txns, txid)
			}
		}
		writeJSON(w, ExplorerHashGET{
			HashType:     "filecontractid",
			Blocks:       blocks,
			Transactions: txns,
		})
		return
	}

	// Try the hash as a siafund output id.
	txids = srv.explorer.SiafundOutputID(types.SiafundOutputID(hash))
	if len(txids) != 0 {
		var txns []types.TransactionID
		var blocks []types.BlockID
		for _, txid := range txids {
			block, _, exists := srv.explorer.Transaction(txid)
			if !exists && build.DEBUG {
				panic("explorer pointing to nonexistant txn")
			}
			if types.TransactionID(block.ID()) == txid {
				blocks = append(blocks, block.ID())
			} else {
				txns = append(txns, txid)
			}
		}
		writeJSON(w, ExplorerHashGET{
			HashType:     "siafundoutputid",
			Blocks:       blocks,
			Transactions: txns,
		})
		return
	}

	// Hash not found, return an error.
	writeError(w, "unrecognized hash used as input to /explorer/hash", http.StatusBadRequest)
}

// explorerHashHandler handles API calls to /explorer/hash.
func (srv *Server) explorerHashHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method == "" || req.Method == "GET" {
		srv.explorerHashHandlerGET(w, req)
		return
	}
	writeError(w, "unrecognized method when calling /explorer/hash", http.StatusBadRequest)
}
