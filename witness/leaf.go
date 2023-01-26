package witness

// TODO: use everywhere where prepareAccountLeafRows is called

func prepareAccountLeaf(leafS, leafC []byte, key []byte, nonExistingAccountProof, noLeaf bool) ([][]byte, [][]byte) {
	var leafRows [][]byte
	var leafForHashing [][]byte
	// When generating a proof that account doesn't exist, the length of both proofs is the same (doesn't reach
	// this code).
	keyRowS, keyRowC, nonExistingAccountRow, nonceBalanceRowS, nonceBalanceRowC, storageCodeHashRowS, storageCodeHashRowC :=
		prepareAccountLeafRows(leafS, leafC, key, nonExistingAccountProof, noLeaf)
	leafRows = append(leafRows, keyRowS)
	leafRows = append(leafRows, keyRowC)
	leafRows = append(leafRows, nonExistingAccountRow) // not really needed
	leafRows = append(leafRows, nonceBalanceRowS)
	leafRows = append(leafRows, nonceBalanceRowC)
	leafRows = append(leafRows, storageCodeHashRowS)
	leafRows = append(leafRows, storageCodeHashRowC)

	leafS = append(leafS, 5)
	leafC = append(leafC, 5)
	leafForHashing = append(leafForHashing, leafS)
	leafForHashing = append(leafForHashing, leafC)

	return leafRows, leafForHashing
}

func prepareStorageLeaf(leafS []byte, key []byte, nonExistingAccountProof bool) ([][]byte, [][]byte) {
	var leafForHashing [][]byte
	leafRows, leafForHashingS := prepareStorageLeafRows(leafS, 2, false)
	leafForHashing = append(leafForHashing, leafForHashingS)

	return leafRows, leafForHashing
}

func prepareStorageLeafRows(row []byte, typ byte, valueIsZero bool) ([][]byte, []byte) {
	// Avoid directly changing the row as it might introduce some bugs later on.
	leaf1 := make([]byte, rowLen)
	leaf2 := make([]byte, rowLen)
	typ2 := byte(13)
	if typ == 3 {
		typ2 = 14
	}
	if len(row) < 32 { // the node doesn't get hashed in this case
		// 192 + 32 = 224
		if row[1] < 128 {
			// last level: [194,32,1]
			// or
			// only one nibble in a leaf (as soon as the leaf has two nibbles, row[1] will have 128 + length)
			// [194,48,1] - this one contains nibble 0 = 48 - 48
			leaf1[0] = row[0]
			leaf1[1] = row[1]
			copy(leaf2, row[2:])
		} else {
			// [196,130,32,0,1]
			keyLen := row[1] - 128
			copy(leaf1, row[:keyLen+2])
			copy(leaf2, row[keyLen+2:])
		}
		leaf1 = append(leaf1, typ)
		leaf2 = append(leaf2, typ2)

		leafForHashing := make([]byte, len(row))
		leafForHashing = append(leafForHashing, 5) // not needed in this case
		return [][]byte{leaf1, leaf2}, leafForHashing
	}	
	if row[0] == 248 {
		// [248,67,160,59,138,106,70,105,186,37,13,38,205,122,69,158,202,157,33,95,131,7,227,58,235,229,3,121,188,90,54,23,236,52,68,161,160,...
		keyLen := row[2] - 128
		copy(leaf1, row[:keyLen+3])
		// there are two RLP meta data bytes which are put in s_rlp1 and s_rlp2,
		// value starts in s_advices[0]
		if !valueIsZero {
			copy(leaf2, row[keyLen+3:]) // RLP data in s_rlp1 and s_rlp2, value starts in s_advices[0]
		}
	} else {
		if row[1] < 128 {
			// last level:
			// [227,32,161,160,187,239,170,18,88,1,56,188,38,60,149,117,120,38,223,78,36,235,129,201,170,170,170,170,170,170,170,170,170,170,170,170]
			// one nibble:
			// [227,48,161,160,187,239,170,18,88,1,56,188,38,60,149,117,120,38,223,78,36,235,129,201,170,170,170,170,170,170,170,170,170,170,170,170]
			leaf1[0] = row[0]
			leaf1[1] = row[1]
			copy(leaf2, row[2:])
		} else {
			// [226,160,59,138,106,70,105,186,37,13,38[227,32,161,160,187,239,170,18,88,1,56,188,38,60,149,117,120,38,223,78,36,235,129,201,170,170,170,170,170,170,170,170,170,170,170,170]
			keyLen := row[1] - 128
			copy(leaf1, row[:keyLen+2])
			if !valueIsZero {
				copy(leaf2, row[keyLen+2:]) // value starts in s_rlp1
			}
		}
	}
	leaf1 = append(leaf1, typ)
	leaf2 = append(leaf2, typ2)

	leafForHashing := make([]byte, len(row))
	copy(leafForHashing, row)
	leafForHashing = append(leafForHashing, 5)

	return [][]byte{leaf1, leaf2}, leafForHashing
}

func prepareEmptyNonExistingStorageRow() []byte {	
	// nonExistingStorageRow is used only for proof that nothing is stored at a particular storage key
	nonExistingStorageRow := make([]byte, rowLen)
	nonExistingStorageRow = append(nonExistingStorageRow, 19)

	return nonExistingStorageRow
}

func prepareNonExistingStorageRow(leafC, keyNibbles []byte, noLeaf bool) []byte {	
	// nonExistingStorageRow is used only for proof that nothing is stored at a particular storage key
	nonExistingStorageRow := prepareEmptyNonExistingStorageRow()
	
	start := 2
	if leafC[0] == 248 {
		start = 3
	}
	keyLenC := int(leafC[start-1]) - 128
	keyRowC := make([]byte, rowLen)
	for i := 0; i < start+keyLenC; i++ {
		keyRowC[i] = leafC[i]
	}

	offset := 0	
	nibblesNum := (keyLenC - 1) * 2
	nonExistingStorageRow[start-1] = leafC[start-1] // length
	if keyRowC[start] != 32 { // odd number of nibbles
		nibblesNum = nibblesNum + 1
		nonExistingStorageRow[start] = keyNibbles[64 - nibblesNum] + 48 
		offset = 1
	} else {
		nonExistingStorageRow[start] = 32
	}
	// Get the last nibblesNum of address:
	remainingNibbles := keyNibbles[64 - nibblesNum:64] // exclude the last one as it is not a nibble
	for i := 0; i < keyLenC-1; i++ {
		nonExistingStorageRow[start+1+i] = remainingNibbles[2*i + offset] * 16 + remainingNibbles[2*i+1 + offset]
	}

	if !noLeaf {
		nonExistingStorageRow[0] = 1 // whether it is wrong leaf
	}

	return nonExistingStorageRow
}

// getNonceBalanceRow takes GetProof account leaf and prepares a row that contains nonce and balance.
func getNonceBalanceRow(leaf []byte, keyLen int) ([]byte, int) {
	nonceStart := 3 + keyLen + 1 + 1 + 1 + 1

	var nonceRlpLen byte
	var balanceStart int
	var nonce []byte

	// If the first nonce byte is > 128, it means it presents (nonce_len - 128),
	// if the first nonce byte is <= 128, the actual nonce value is < 128 and is exactly this first byte
	// (however, when nonce = 0, the actual value that is stored is 128)
	if leaf[nonceStart] <= 128 {
		// only one nonce byte
		nonceRlpLen = 1
		nonce = leaf[nonceStart : nonceStart+int(nonceRlpLen)]
		balanceStart = nonceStart + int(nonceRlpLen)
	} else {
		nonceRlpLen = leaf[nonceStart] - 128
		nonce = leaf[nonceStart : nonceStart+int(nonceRlpLen)+1]
		balanceStart = nonceStart + int(nonceRlpLen) + 1
	}

	var balanceRlpLen byte
	var storageStart int
	if leaf[balanceStart] <= 128 {
		// only one balance byte
		balanceRlpLen = 1
		storageStart = balanceStart + int(balanceRlpLen)
	} else {
		balanceRlpLen = leaf[balanceStart] - 128
		storageStart = balanceStart + int(balanceRlpLen) + 1
	}

	nonceBalanceRow := make([]byte, rowLen)
	for i := 0; i < len(nonce); i++ {
		nonceBalanceRow[branchNodeRLPLen+i] = nonce[i]
	}
	nonceBalanceRow[0] = leaf[3+keyLen]
	nonceBalanceRow[1] = leaf[3+keyLen+1]
	nonceBalanceRow[branch2start] = leaf[3+keyLen+1+1]
	nonceBalanceRow[branch2start+1] = leaf[3+keyLen+1+1+1]
	var balance []byte
	if balanceRlpLen == 1 {
		balance = leaf[balanceStart : balanceStart+int(balanceRlpLen)]
	} else {
		balance = leaf[balanceStart : balanceStart+int(balanceRlpLen)+1]
	}
	for i := 0; i < len(balance); i++ {
		nonceBalanceRow[branch2start+2+i] = balance[i] // c_advices
	}

	return nonceBalanceRow, storageStart
}

// getStorageRootCodeHashRow takes GetProof account leaf and prepares a row that contains storage root and hash root.
func getStorageRootCodeHashRow(leaf []byte, storageStart int) []byte {
	storageCodeHashRow := make([]byte, rowLen)
	storageRlpLen := leaf[storageStart] - 128
	if storageRlpLen != 32 {
		panic("Account leaf RLP 3")
	}
	storage := leaf[storageStart : storageStart+32+1]
	for i := 0; i < 33; i++ {
		storageCodeHashRow[branchNodeRLPLen-1+i] = storage[i]
	}
	codeHashStart := storageStart + int(storageRlpLen) + 1
	codeHashRlpLen := leaf[codeHashStart] - 128
	if codeHashRlpLen != 32 {
		panic("Account leaf RLP 4")
	}
	codeHash := leaf[codeHashStart : codeHashStart+32+1]
	for i := 0; i < 33; i++ {
		storageCodeHashRow[branch2start+1+i] = codeHash[i] // start from c_rlp2
	}

	return storageCodeHashRow
}

func prepareAccountLeafRows(leafS, leafC, addressNibbles []byte, nonExistingAccountProof, noLeaf bool) ([]byte, []byte, []byte, []byte, []byte, []byte, []byte) {	
	// wrongLeaf has a meaning only for non existing account proof. For this proof, there are two cases:
	// 1. A leaf is returned that is not at the required address (wrong leaf).
	// 2. A branch is returned as the last element of getProof and
	//    there is nil object at address position. Placeholder account leaf is added in this case.
	keyLenS := int(leafS[2]) - 128
	keyLenC := int(leafC[2]) - 128
	keyRowS := make([]byte, rowLen)
	keyRowC := make([]byte, rowLen)
	for i := 0; i < 3+keyLenS; i++ {
		keyRowS[i] = leafS[i]
	}
	for i := 0; i < 3+keyLenC; i++ {
		keyRowC[i] = leafC[i]
	}

	// For non existing account proof, keyRowS (=keyRowC in this case) stores the key of
	// the wrong leaf. We store the key of the required leaf (which doesn't exist)
	// in nonExistingAccountRow.

	// nonExistingAccountRow is used only for proof that account doesn't exist
	nonExistingAccountRow := make([]byte, rowLen)
	nonExistingAccountRow = append(nonExistingAccountRow, 18)
	
	offset := 0	
	nibblesNum := (keyLenC - 1) * 2
	nonExistingAccountRow[2] = leafC[2] // length
	if keyRowC[3] != 32 { // odd number of nibbles
		nibblesNum = nibblesNum + 1
		nonExistingAccountRow[3] = addressNibbles[64 - nibblesNum] + 48 
		offset = 1
	} else {
		nonExistingAccountRow[3] = 32
	}
	// Get the last nibblesNum of address:
	remainingNibbles := addressNibbles[64 - nibblesNum:64] // exclude the last one as it is not a nibble
	for i := 0; i < keyLenC-1; i++ {
		nonExistingAccountRow[4+i] = remainingNibbles[2*i + offset] * 16 + remainingNibbles[2*i+1 + offset]
	}

	// for non-existing account proof we have leafS = leafC
	if nonExistingAccountProof && !noLeaf {
		nonExistingAccountRow[0] = 1 // whether it is wrong leaf
	}
	
	nonceBalanceRowS := make([]byte, rowLen)
	nonceBalanceRowC := make([]byte, rowLen)
	storageCodeHashRowS := make([]byte, rowLen)
	storageCodeHashRowC := make([]byte, rowLen)

	if !noLeaf {
		rlpStringSecondPartLenS := leafS[3+keyLenS] - 183
		if rlpStringSecondPartLenS != 1 {
			panic("Account leaf RLP at this position should be 1 (S)")
		}
		rlpStringSecondPartLenC := leafC[3+keyLenC] - 183
		if rlpStringSecondPartLenC != 1 {
			panic("Account leaf RLP at this position should be 1 (C)")
		}
		rlpStringLenS := leafS[3+keyLenS+1]
		rlpStringLenC := leafC[3+keyLenC+1]

		// [248,112,157,59,158,160,175,159,65,212,107,23,98,208,38,205,150,63,244,2,185,236,246,95,240,224,191,229,27,102,202,231,184,80,248,78
		// In this example RLP, there are first 36 bytes of a leaf.
		// 157 means there are 29 bytes for key (157 - 128).
		// Positions 32-35: 184, 80, 248, 78.
		// 184 - 183 = 1 means length of the second part of a string.
		// 80 means length of a string.
		// 248 - 247 = 1 means length of the second part of a list.
		// 78 means length of a list.

		rlpListSecondPartLenS := leafS[3+keyLenS+1+1] - 247
		if rlpListSecondPartLenS != 1 {
			panic("Account leaf RLP 1 (S)")
		}
		rlpListSecondPartLenC := leafC[3+keyLenC+1+1] - 247
		if rlpListSecondPartLenC != 1 {
			panic("Account leaf RLP 1 (C)")
		}

		rlpListLenS := leafS[3+keyLenS+1+1+1]
		if rlpStringLenS != rlpListLenS+2 {
			panic("Account leaf RLP 2 (S)")
		}

		rlpListLenC := leafC[3+keyLenC+1+1+1]
		if rlpStringLenC != rlpListLenC+2 {
			panic("Account leaf RLP 2 (C)")
		}

		storageStartS := 0
		storageStartC := 0
		nonceBalanceRowS, storageStartS = getNonceBalanceRow(leafS, keyLenS)
		nonceBalanceRowC, storageStartC = getNonceBalanceRow(leafC, keyLenC)

		storageCodeHashRowS = getStorageRootCodeHashRow(leafS, storageStartS)
		storageCodeHashRowC = getStorageRootCodeHashRow(leafC, storageStartC)
	} 

	keyRowS = append(keyRowS, 6)
	keyRowC = append(keyRowC, 4)
	nonceBalanceRowS = append(nonceBalanceRowS, 7)
	nonceBalanceRowC = append(nonceBalanceRowC, 8)
	storageCodeHashRowS = append(storageCodeHashRowS, 9)
	storageCodeHashRowC = append(storageCodeHashRowC, 11)

	return keyRowS, keyRowC, nonExistingAccountRow, nonceBalanceRowS, nonceBalanceRowC, storageCodeHashRowS, storageCodeHashRowC
}