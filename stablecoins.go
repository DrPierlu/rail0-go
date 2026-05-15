package rail0

// StablecoinInfo holds static metadata for a single stablecoin on a specific chain.
type StablecoinInfo struct {
	// Address is the checksummed contract address.
	Address string
	// Decimals is the number of decimal places (typically 6 for USDC/USDT, 18 for DAI).
	Decimals int
	// EIP3009 is true when the token supports transferWithAuthorization (Circle standard). Required by RAIL0.
	EIP3009 bool
	// EIP2612 is true when the token supports permit. Not used by RAIL0.
	EIP2612 bool
	// Bridged is true when the token is a bridge-wrapped variant that may lack auth extensions.
	Bridged bool
}

// ChainStablecoins groups a chain's metadata with its token registry.
type ChainStablecoins struct {
	ChainID int
	Tokens  map[string]StablecoinInfo
}

// Stablecoins is the registry of known stablecoin addresses and capabilities across supported EVM chains.
var Stablecoins = map[string]ChainStablecoins{
	"ethereum": {
		ChainID: 1,
		Tokens: map[string]StablecoinInfo{
			"USDC":  {Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Decimals: 6, EIP3009: true},
			"EURC":  {Address: "0x1aBaEA1f7C830bD89Acc67eC4af516284b1bC33c", Decimals: 6, EIP3009: true},
			"PYUSD": {Address: "0x6c3ea9036406852006290770BEdFcAbA0e23A0e8", Decimals: 6, EIP3009: true},
			"USDT":  {Address: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Decimals: 6},
			"DAI":   {Address: "0x6B175474E89094C44Da98b954EedeAC495271d0F", Decimals: 18, EIP2612: true},
		},
	},
	"base": {
		ChainID: 8453,
		Tokens: map[string]StablecoinInfo{
			"USDC":  {Address: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", Decimals: 6, EIP3009: true},
			"EURC":  {Address: "0x60a3E35Cc302bFA44Cb288Bc5a4F316Fdb1adb42", Decimals: 6, EIP3009: true},
			"USDbC": {Address: "0xd9aAEc86B65D86f6A7B5B1b0c42FFA531710b6CA", Decimals: 6, Bridged: true},
		},
	},
	"polygon": {
		ChainID: 137,
		Tokens: map[string]StablecoinInfo{
			"USDC":   {Address: "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359", Decimals: 6, EIP3009: true},
			"USDC.e": {Address: "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174", Decimals: 6, EIP3009: true, Bridged: true},
			"USDT":   {Address: "0xc2132D05D31c914a87C6611C10748AEb04B58e8F", Decimals: 6},
			"DAI":    {Address: "0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063", Decimals: 18},
		},
	},
	"arbitrumOne": {
		ChainID: 42161,
		Tokens: map[string]StablecoinInfo{
			"USDC":   {Address: "0xaf88d065e77c8cC2239327C5EDb3A432268e5831", Decimals: 6, EIP3009: true},
			"USDC.e": {Address: "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8", Decimals: 6, Bridged: true},
			"USDT":   {Address: "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9", Decimals: 6},
			"DAI":    {Address: "0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1", Decimals: 18, EIP2612: true},
		},
	},
	"optimism": {
		ChainID: 10,
		Tokens: map[string]StablecoinInfo{
			"USDC":   {Address: "0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85", Decimals: 6, EIP3009: true},
			"USDC.e": {Address: "0x7F5c764cBc14f9669B88837ca1490cCa17c31607", Decimals: 6, Bridged: true},
			"USDT":   {Address: "0x94b008aA00579c1307B0EF2c499aD98a8ce58e58", Decimals: 6},
			"DAI":    {Address: "0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1", Decimals: 18, EIP2612: true},
		},
	},
	"avalanche": {
		ChainID: 43114,
		Tokens: map[string]StablecoinInfo{
			"USDC":   {Address: "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E", Decimals: 6, EIP3009: true},
			"USDC.e": {Address: "0xA7D7079b0FEaD91F3e65f86E8915Cb59c1a4C664", Decimals: 6, Bridged: true},
			"USDT":   {Address: "0x9702230A8Ea53601f5cD2dc00fDBc13d4dF4A8c7", Decimals: 6},
		},
	},
	"celo": {
		ChainID: 42220,
		Tokens: map[string]StablecoinInfo{
			"USDC": {Address: "0xcebA9300f2b948710d2De3250b7Ad3e4aFb0e50a", Decimals: 6, EIP3009: true},
			"cUSD": {Address: "0x765DE816845861e75A25fCA122bb6898B8B1282a", Decimals: 18, EIP3009: true},
			"cEUR": {Address: "0xD8763CBa276a3738E6DE85b4b3bF5FDed6D6cA73", Decimals: 18, EIP3009: true},
		},
	},
}

// StablecoinToken is a simplified token view returned by EIP3009Tokens and EIP2612Tokens.
type StablecoinToken struct {
	Symbol   string
	Address  string
	Decimals int
}

// ChainInfo returns the registry entry for a chain and whether it was found.
// Supported chain names: "ethereum", "base", "polygon", "arbitrumOne", "optimism", "avalanche", "celo".
func ChainInfo(chain string) (ChainStablecoins, bool) {
	c, ok := Stablecoins[chain]
	return c, ok
}

// EIP3009Tokens returns all tokens on the given chain that support transferWithAuthorization (EIP-3009).
// These are the tokens compatible with RAIL0.
func EIP3009Tokens(chain string) []StablecoinToken {
	c, ok := Stablecoins[chain]
	if !ok {
		return nil
	}
	var out []StablecoinToken
	for symbol, t := range c.Tokens {
		if t.EIP3009 {
			out = append(out, StablecoinToken{Symbol: symbol, Address: t.Address, Decimals: t.Decimals})
		}
	}
	return out
}

// EIP2612Tokens returns all tokens on the given chain that support permit (EIP-2612).
func EIP2612Tokens(chain string) []StablecoinToken {
	c, ok := Stablecoins[chain]
	if !ok {
		return nil
	}
	var out []StablecoinToken
	for symbol, t := range c.Tokens {
		if t.EIP2612 {
			out = append(out, StablecoinToken{Symbol: symbol, Address: t.Address, Decimals: t.Decimals})
		}
	}
	return out
}
