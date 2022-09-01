f = ''
for i in range(258):
    # f += f's{i},'
    # f += f'ps{i},'
    # f += f'ps{i}[i],'
    f += f'circuit.TrieProofsSenderBefore[i][{i}][:],\n'
    # f += f'circuit.TrieProofsReceiverBefore[i][{i}][:],\n'
    # f += f't.TrieProofsSenderBefore[0][{i}][:],\n'
    # f += f't.TrieProofsReceiverAfter[0][{i}][:],\n'
    # f += f's{i} := api.Select(api.IsZero(api.Sub(paths[i-1], {i})), h, ps{i}[i])\n'
    # if i % 16 == 15:
    #     f += '\n'
print(f)
