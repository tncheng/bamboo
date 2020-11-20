package blockchain

import (
	"fmt"
	"github.com/gitferry/bamboo/crypto"
	"github.com/gitferry/bamboo/types"
)

type BlockChain struct {
	forrest          *LevelledForest
	quorum           *Quorum
	longestTailBlock *Block
	// measurement
	highestComitted       int
	committedBlocks       int
	honestCommittedBlocks int
}

func NewBlockchain(n int) *BlockChain {
	bc := new(BlockChain)
	bc.forrest = NewLevelledForest()
	bc.quorum = NewQuorum(n)
	return bc
}

func (bc *BlockChain) AddBlock(block *Block) {
	blockContainer := &BlockContainer{block}
	// TODO: add checks
	bc.forrest.AddVertex(blockContainer)
}

func (bc *BlockChain) AddVote(vote *Vote) (bool, *QC) {
	return bc.quorum.Add(vote)
}

func (bc *BlockChain) CalForkingRate() float32 {
	var forkingRate float32
	//if bc.Height == 0 {
	//	return 0
	//}
	//total := 0
	//for i := 1; i <= bc.Height; i++ {
	//	total += len(bc.Blocks[i])
	//}
	//
	//forkingrate := float32(bc.Height) / float32(total)
	return forkingRate
}

func (bc *BlockChain) GetBlockByID(id crypto.Identifier) (*Block, error) {
	vertex, exists := bc.forrest.GetVertex(id)
	if !exists {
		return nil, fmt.Errorf("the block does not exist, id: %x", id)
	}
	return vertex.GetBlock(), nil
}

func (bc *BlockChain) GetParentBlock(id crypto.Identifier) (*Block, error) {
	vertex, exists := bc.forrest.GetVertex(id)
	if !exists {
		return nil, fmt.Errorf("the block does not exist, id: %x", id)
	}
	parentID, _ := vertex.Parent()
	parentVertex, exists := bc.forrest.GetVertex(parentID)
	if !exists {
		return nil, fmt.Errorf("parent block does not exist, id: %x", parentID)
	}
	return parentVertex.GetBlock(), nil
}

func (bc *BlockChain) GetGrandParentBlock(id crypto.Identifier) (*Block, error) {
	parentBlock, err := bc.GetParentBlock(id)
	if err != nil {
		return nil, fmt.Errorf("cannot get parent block: %w", err)
	}
	return bc.GetParentBlock(parentBlock.ID)
}

// CommitBlock prunes blocks and returns committed blocks up to the last committed one and prunedBlocks
func (bc *BlockChain) CommitBlock(id crypto.Identifier) ([]*Block, []*Block, error) {
	vertex, ok := bc.forrest.GetVertex(id)
	if !ok {
		return nil, nil, fmt.Errorf("cannot find the block, id: %x", id)
	}
	committedView := vertex.GetBlock().View
	bc.highestComitted = int(vertex.GetBlock().View)
	var committedBlocks []*Block
	for block := vertex.GetBlock(); uint64(block.View) > bc.forrest.LowestLevel; {
		committedBlocks = append(committedBlocks, block)
		_, ok := bc.quorum.votes[block.ID]
		if ok {
			delete(bc.quorum.votes, block.ID)
		}
		bc.committedBlocks++
		vertex, exists := bc.forrest.GetVertex(block.PrevID)
		if !exists {
			break
		}
		block = vertex.GetBlock()
	}
	prunedBlocks, err := bc.forrest.PruneUpToLevel(uint64(committedView))
	if err != nil {
		return nil, nil, fmt.Errorf("cannot prune the blockchain to the committed block, id: %w", err)
	}

	return committedBlocks, bc.removeCommittedBlocks(committedBlocks, prunedBlocks), nil
}

func (bc *BlockChain) removeCommittedBlocks(committed []*Block, pruned []*Block) []*Block {
	if len(pruned) == 1 || len(committed) >= len(pruned) {
		return nil
	}
	for _, cb := range committed {
		for i, pb := range pruned {
			if cb.ID == pb.ID {
				pruned = append(pruned[:i], pruned[i+1:]...)
				break
			}
		}
	}
	return pruned[1:]
}

func (bc *BlockChain) GetChildrenBlocks(id crypto.Identifier) []*Block {
	var blocks []*Block
	iterator := bc.forrest.GetChildren(id)
	for I := iterator; I.HasNext(); {
		blocks = append(blocks, I.NextVertex().GetBlock())
	}
	return blocks
}

func (bc *BlockChain) GetChainGrowth() float64 {
	return float64(bc.honestCommittedBlocks) / float64(bc.highestComitted)
}

func (bc *BlockChain) GetChainQuality() float64 {
	return float64(bc.honestCommittedBlocks) / float64(bc.committedBlocks)
}

func (bc *BlockChain) GetHighestComitted() int {
	return bc.highestComitted
}

func (bc *BlockChain) GetHonestCommittedBlocks() int {
	return bc.honestCommittedBlocks
}

func (bc *BlockChain) GetCommittedBlocks() int {
	return bc.committedBlocks
}

func (bc *BlockChain) GetBlockByView(view types.View) *Block {
	iterator := bc.forrest.GetVerticesAtLevel(uint64(view))
	return iterator.next.GetBlock()
}
