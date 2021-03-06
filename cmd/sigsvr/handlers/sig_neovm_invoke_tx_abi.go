
package handlers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	clisvrcom "github.com/mixbee/mixbee/cmd/sigsvr/common"
	cliutil "github.com/mixbee/mixbee/cmd/utils"
	"github.com/mixbee/mixbee/common"
	"github.com/mixbee/mixbee/common/log"
	httpcom "github.com/mixbee/mixbee/http/base/common"
)

type SigNeoVMInvokeTxAbiReq struct {
	GasPrice    uint64          `json:"gas_price"`
	GasLimit    uint64          `json:"gas_limit"`
	Address     string          `json:"address"`
	Method      string          `json:"method"`
	Params      []string        `json:"params"`
	ContractAbi json.RawMessage `json:"contract_abi"`
}

type SigNeoVMInvokeTxAbiRsp struct {
	SignedTx string `json:"signed_tx"`
}

func SigNeoVMInvokeAbiTx(req *clisvrcom.CliRpcRequest, resp *clisvrcom.CliRpcResponse) {
	rawReq := &SigNeoVMInvokeTxAbiReq{}
	err := json.Unmarshal(req.Params, rawReq)
	if err != nil {
		log.Infof("SigNeoVMInvokeAbiTx json.Unmarshal SigNeoVMInvokeTxAbiReq:%s error:%s", req.Params, err)
		resp.ErrorCode = clisvrcom.CLIERR_INVALID_PARAMS
		return
	}
	contractAbi, err := cliutil.NewNeovmContractAbi(rawReq.ContractAbi)
	if err != nil {
		resp.ErrorCode = clisvrcom.CLIERR_ABI_UNMATCH
		resp.ErrorInfo = err.Error()
		return
	}
	funcAbi := contractAbi.GetFunc(rawReq.Method)
	if funcAbi == nil {
		resp.ErrorCode = clisvrcom.CLIERR_ABI_NOT_FOUND
		return
	}
	invokParams, err := cliutil.ParseNeovmFunc(rawReq.Params, funcAbi)
	if err != nil {
		resp.ErrorCode = clisvrcom.CLIERR_ABI_UNMATCH
		resp.ErrorInfo = err.Error()
		return
	}
	contAddr, err := common.AddressFromHexString(rawReq.Address)
	if err != nil {
		log.Infof("Cli Qid:%s SigNeoVMInvokeAbiTx AddressParseFromBytes:%s error:%s", req.Qid, rawReq.Address, err)
		resp.ErrorCode = clisvrcom.CLIERR_INVALID_PARAMS
		return
	}
	tx, err := httpcom.NewNeovmInvokeTransaction(rawReq.GasPrice, rawReq.GasLimit, contAddr, invokParams)
	if err != nil {
		log.Infof("Cli Qid:%s SigNeoVMInvokeAbiTx InvokeNeoVMContractTx error:%s", req.Qid, err)
		resp.ErrorCode = clisvrcom.CLIERR_INVALID_PARAMS
		return
	}
	signer := clisvrcom.DefAccount
	err = cliutil.SignTransaction(signer, tx)
	if err != nil {
		log.Infof("Cli Qid:%s SigNeoVMInvokeAbiTx SignTransaction error:%s", req.Qid, err)
		resp.ErrorCode = clisvrcom.CLIERR_INTERNAL_ERR
		return
	}
	buf := bytes.NewBuffer(nil)
	err = tx.Serialize(buf)
	if err != nil {
		log.Infof("Cli Qid:%s SigNeoVMInvokeAbiTx tx Serialize error:%s", req.Qid, err)
		resp.ErrorCode = clisvrcom.CLIERR_INTERNAL_ERR
		return
	}
	resp.Result = &SigNeoVMInvokeTxAbiRsp{
		SignedTx: hex.EncodeToString(buf.Bytes()),
	}
}
