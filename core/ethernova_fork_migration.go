package core

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params/ethernova"
	"github.com/ethereum/go-ethereum/params/types/coregeth"
	"github.com/ethereum/go-ethereum/params/types/ctypes"
)

func ethernovaPatchConfigIfNeeded(cfg ctypes.ChainConfigurator, head uint64) (bool, error) {
	if cfg == nil {
		return false, nil
	}
	chainID := cfg.GetChainID()
	if chainID == nil || chainID.Uint64() != ethernova.NewChainID {
		return false, nil
	}

	updated := false

	forkBlock := ethernova.EVMCompatibilityForkBlock
	missing, mismatched, err := ethernovaForkStatus(cfg, forkBlock)
	if err != nil {
		return false, err
	}
	if len(mismatched) > 0 {
		return false, fmt.Errorf("ethernova chain config has unexpected fork block values (%s); expected %d", strings.Join(mismatched, ", "), forkBlock)
	}
	if missing {
		if head >= forkBlock {
			return false, fmt.Errorf("UPGRADE REQUIRED: ethernova chain config missing Constantinople/Petersburg/Istanbul fork blocks; head=%d fork=%d. Refusing to start; upgrade before block %d", head, forkBlock, forkBlock)
		}
		updatedForks, err := ethernovaApplyForks(cfg, forkBlock)
		if err != nil {
			return false, err
		}
		if updatedForks {
			log.Warn("Ethernova chain config upgraded in-place", "fork_block", forkBlock, "head", head, "feature", "evm-compat")
		}
		updated = updated || updatedForks
	}

	eip658Block := ethernova.EIP658ForkBlock
	missing, mismatched, err = ethernovaEIP658Status(cfg, eip658Block)
	if err != nil {
		return updated, err
	}
	if len(mismatched) > 0 {
		return updated, fmt.Errorf("ethernova chain config has unexpected EIP-658 fork block values (%s); expected %d", strings.Join(mismatched, ", "), eip658Block)
	}
	if !missing {
		return updated, nil
	}
	if head >= eip658Block {
		return updated, fmt.Errorf("UPGRADE REQUIRED: ethernova chain config missing EIP-658 fork block; head=%d fork=%d. Refusing to start; upgrade before block %d", head, eip658Block, eip658Block)
	}
	updated658, err := ethernovaApplyEIP658(cfg, eip658Block)
	if err != nil {
		return updated, err
	}
	if updated658 {
		log.Warn("Ethernova chain config upgraded in-place", "fork_block", eip658Block, "head", head, "feature", "eip658")
	}
	return updated || updated658, nil
}

func ethernovaForkStatus(cfg ctypes.ChainConfigurator, forkBlock uint64) (missing bool, mismatched []string, err error) {
	cg, ok := cfg.(*coregeth.CoreGethChainConfig)
	if !ok {
		return false, nil, fmt.Errorf("unsupported chain config type for ethernova: %T", cfg)
	}
	checkBig := func(name string, val *big.Int) {
		if val == nil {
			missing = true
			return
		}
		if val.Uint64() != forkBlock {
			mismatched = append(mismatched, fmt.Sprintf("%s=%d", name, val.Uint64()))
		}
	}
	checkBig("constantinopleBlock", cg.ConstantinopleBlock)
	checkBig("petersburgBlock", cg.PetersburgBlock)
	checkBig("istanbulBlock", cg.IstanbulBlock)

	check := func(name string, val *uint64) {
		if val == nil {
			return
		}
		if *val != forkBlock {
			mismatched = append(mismatched, fmt.Sprintf("%s=%d", name, *val))
		}
	}
	check("eip145", cfg.GetEIP145Transition())
	check("eip1014", cfg.GetEIP1014Transition())
	check("eip1052", cfg.GetEIP1052Transition())
	check("eip1283", cfg.GetEIP1283Transition())
	check("petersburg", cfg.GetEIP1283DisableTransition())
	check("eip152", cfg.GetEIP152Transition())
	check("eip1108", cfg.GetEIP1108Transition())
	check("eip1344", cfg.GetEIP1344Transition())
	check("eip1884", cfg.GetEIP1884Transition())
	check("eip2028", cfg.GetEIP2028Transition())
	check("eip2200", cfg.GetEIP2200Transition())
	return missing, mismatched, nil
}

func ethernovaEIP658Status(cfg ctypes.ChainConfigurator, forkBlock uint64) (missing bool, mismatched []string, err error) {
	cg, ok := cfg.(*coregeth.CoreGethChainConfig)
	if !ok {
		return false, nil, fmt.Errorf("unsupported chain config type for ethernova: %T", cfg)
	}
	if cg.EIP658FBlock == nil {
		return true, nil, nil
	}
	if cg.EIP658FBlock.Uint64() != forkBlock {
		mismatched = append(mismatched, fmt.Sprintf("eip658FBlock=%d", cg.EIP658FBlock.Uint64()))
	}
	return false, mismatched, nil
}

func ethernovaApplyForks(cfg ctypes.ChainConfigurator, forkBlock uint64) (bool, error) {
	cg, ok := cfg.(*coregeth.CoreGethChainConfig)
	if !ok {
		return false, fmt.Errorf("unsupported chain config type for ethernova: %T", cfg)
	}
	updated := false
	if cg.ConstantinopleBlock == nil {
		cg.ConstantinopleBlock = new(big.Int).SetUint64(forkBlock)
		updated = true
	}
	if cg.PetersburgBlock == nil {
		cg.PetersburgBlock = new(big.Int).SetUint64(forkBlock)
		updated = true
	}
	if cg.IstanbulBlock == nil {
		cg.IstanbulBlock = new(big.Int).SetUint64(forkBlock)
		updated = true
	}
	return updated, nil
}

func ethernovaApplyEIP658(cfg ctypes.ChainConfigurator, forkBlock uint64) (bool, error) {
	cg, ok := cfg.(*coregeth.CoreGethChainConfig)
	if !ok {
		return false, fmt.Errorf("unsupported chain config type for ethernova: %T", cfg)
	}
	if cg.EIP658FBlock != nil {
		return false, nil
	}
	cg.EIP658FBlock = new(big.Int).SetUint64(forkBlock)
	return true, nil
}
