package check

import (
	"context"
	"math"
	"sort"

	"github.com/gernest/8x8/pkg/models"
)

type Board models.Board

func (b *Board) player_pieces(player Player) []*models.Piece {
	if player {
		return b.BlackPieces
	}
	return b.WhitePieces
}
func (b *Board) player_positions(player Player) []int32 {
	if player {
		return b.BlackPositions
	}
	return b.WhitePositions
}

func (b *Board) resetPieces() {
	b.PieceById = make(map[int32]*models.Piece)
	for i := 0; i < len(b.Pieces); i++ {
		p := &Piece{b.Pieces[i]}
		p.reset_for_new_board()
		b.PieceById[p.Id] = p.Piece
	}
	b.buildSearch()
}

func (b *Board) buildSearch() {
	var ls []*models.Piece
	for _, v := range b.Pieces {
		if !v.Captured {
			ls = append(ls, v)
		}
	}
	b.UncapturedPieces = ls
	b.build_filled_positions()
	b.build_open_positions()
	b.build_player_positions()
	b.build_player_pieces()
	b.build_position_pieces()
}

func (b *Board) build_filled_positions() {
	var ls []int32
	for _, v := range b.UncapturedPieces {
		ls = append(ls, v.Position)
	}
	b.FilledPositions = ls
}

func in(ls []int32, v int32) bool {
	for x := 0; x < len(ls); x++ {
		if ls[x] == v {
			return true
		}
	}
	return false
}

func (b *Board) build_open_positions() {
	var ls []int32
	for i := 1; i < PositionCount; i++ {
		if !in(b.FilledPositions, int32(i)) {
			ls = append(ls, int32(i))
		}
	}
	b.OpenPositions = ls
}

func (b *Board) build_player_positions() {
	var black, white []int32
	for _, v := range b.UncapturedPieces {
		switch Player(v.Player) {
		case White:
			white = append(white, v.Position)
		case Black:
			black = append(black, v.Position)
		}
	}
	b.WhitePositions = white
	b.BlackPositions = black
}

func (b *Board) build_player_pieces() {
	var white, black []*models.Piece
	for _, v := range b.UncapturedPieces {
		switch Player(v.Player) {
		case White:
			white = append(white, v)
		case Black:
			black = append(black, v)
		}
	}
	b.WhitePieces = white
	b.BlackPieces = black
}

func (b *Board) build_position_pieces() {
	m := make(map[int32]*models.Piece)
	for _, v := range b.UncapturedPieces {
		m[v.Position] = v
	}
	b.PositionPieces = m
}

func (b *Board) get_pieces_by_player(p Player) []*models.Piece {
	return b.player_pieces(p)
}

func (b *Board) get_positions_by_player(p Player) []int32 {
	return b.player_positions(p)
}

func (b *Board) get_pieces_in_play() []*models.Piece {
	if b.PieceRequiringFurtherCaptureMoves == nil {
		return b.player_pieces(Player(b.PlayertTurn))
	}
	return []*models.Piece{b.PieceRequiringFurtherCaptureMoves}
}

func (b *Board) get_piece_by_position(p int32) *models.Piece {
	return b.PositionPieces[p]
}

func (b *Board) position_is_open(p int32) bool {
	return b.get_piece_by_position(p) == nil
}

func (b *Board) get_possible_moves() []*models.Move {
	capture_moves := b.get_possible_capture_moves()
	if len(capture_moves) > 0 {
		return capture_moves
	}
	return b.get_possible_positional_moves()
}

func (b *Board) deepCopy() *Board {
	return b
}

func (b *Board) get_possible_capture_moves() []*models.Move {
	var moves []*models.Move
	for _, piece := range b.get_pieces_in_play() {
		p := &Piece{piece}
		moves = append(moves, p.get_possible_capture_moves(b)...)
	}
	return moves
}

func (b *Board) get_possible_positional_moves() []*models.Move {
	var moves []*models.Move
	for _, piece := range b.get_pieces_in_play() {
		p := &Piece{piece}
		moves = append(moves, p.get_possible_positional_moves(b)...)
	}
	return moves
}

func (b *Board) perform_positional_move(move *models.Move) {
	b.PreviousMoveWasCapture = false
	b.move_piece(move)
	b.switch_turn()
}

func (b *Board) is_valid_row_and_column(row, column int) bool {
	if row < 0 || row >= Height {
		return false
	}
	if column < 0 || column >= Width {
		return false
	}
	return true
}

func (b *Board) perform_capture_move(move *models.Move) {
	b.PreviousMoveWasCapture = true
	piece := b.get_piece_by_position(move.From)
	originally_was_king := piece.King
	enemy_piece := Piece{b.PieceById[piece.CaptureMoveEnemies[move.To]]}
	enemy_piece.capture()
	b.move_piece(move)
	var further_capture_moves_for_piece []int
	for _, capture_move := range b.get_possible_capture_moves() {
		if move.To == capture_move.From {
			further_capture_moves_for_piece = append(further_capture_moves_for_piece, int(capture_move.From), int(capture_move.To))
		}
	}
	if further_capture_moves_for_piece != nil && originally_was_king == piece.King {
		b.PieceRequiringFurtherCaptureMoves = b.get_piece_by_position(move.To)
	} else {
		b.PieceRequiringFurtherCaptureMoves = nil
		b.switch_turn()
	}
}

func (b *Board) switch_turn() {
	b.PlayertTurn = !b.PlayertTurn
}

func (b *Board) move_piece(move *models.Move) {
	Piece{b.get_piece_by_position(move.From)}.move(move.To)
	sort.Slice(b.Pieces, func(i, j int) bool {
		return b.Pieces[i].Position < b.Pieces[j].Position
	})
}

type boardKey struct{}

func get(ctx context.Context) *Board {
	return ctx.Value(boardKey{}).(*Board)
}

type Player bool

const (
	White Player = false
	Black Player = true
)

type Piece struct {
	*models.Piece
}

func (p Piece) reset_for_new_board() {
	p.PossiblePositionalMoves = nil
	p.PossibleCaptureMoves = nil
}

func (p Piece) capture() {
	p.Captured = true
	p.Position = 0
}

func (p Piece) is_movable(board *Board) bool {
	return (p.get_possible_capture_moves(board) != nil ||
		p.get_possible_positional_moves(board) != nil) && !p.Captured
}

func (p Piece) move(new_position int32) {
	p.Position = new_position
	p.King = p.King || p.is_on_enemy_home_row()
}

func (p Piece) get_possible_capture_moves(board *Board) []*models.Move {
	if p.PossibleCaptureMoves == nil {
		p.PossibleCaptureMoves = p.build_possible_capture_moves(board)
	}
	return p.PossibleCaptureMoves
}

func (p Piece) build_possible_capture_moves(board *Board) []*models.Move {
	var adjacent_enemy_positions []int32
	for _, pos := range p.get_adjacent_positions() {
		pns := board.get_positions_by_player(Player(!p.Player))
		if in(pns, pos) {
			adjacent_enemy_positions = append(adjacent_enemy_positions, pos)
		}
	}
	var capture_move_positions []int32
	for _, enemy_position := range adjacent_enemy_positions {
		enemy_piece := &Piece{board.get_piece_by_position(enemy_position)}
		position_behind_enemy := p.get_position_behind_enemy(enemy_piece)
		if position_behind_enemy != 0 && board.position_is_open(position_behind_enemy) {
			capture_move_positions = append(capture_move_positions, position_behind_enemy)
			p.CaptureMoveEnemies[position_behind_enemy] = enemy_piece.Id
		}
	}
	return p.create_moves_from_new_positions(capture_move_positions)
}

func (p Piece) get_position_behind_enemy(enemy_piece *Piece) int32 {
	current_row := p.get_row()
	current_column := p.get_column()
	enemy_row := enemy_piece.get_row()
	enemy_column := enemy_piece.get_column()
	column_adjustment := int32(1)
	if current_row%2 == 0 {
		column_adjustment = -1
	}
	column_behind_enemy := enemy_column
	if current_column == enemy_column {
		column_behind_enemy = current_column + column_adjustment
	}
	row_behind_enemy := enemy_row + (enemy_row - current_row)
	return layout[row_behind_enemy][column_behind_enemy]
}

func (p Piece) get_column() int32 {
	return (p.Position - 1) % Width
}

func (p Piece) get_row() int32 {
	return p.get_row_from_position(p.Position)
}

func (p Piece) get_row_from_position(pos int32) int32 {
	return int32(math.Ceil(float64(pos)/Width)) - 1
}

func (p Piece) is_on_enemy_home_row() bool {
	pos := PositionCount
	if Player(!p.Player) == White {
		pos = 1
	}
	return p.get_row() == p.get_row_from_position(int32(pos))
}

func (p Piece) get_possible_positional_moves(board *Board) (o []*models.Move) {
	if p.PossiblePositionalMoves == nil {
		p.PossiblePositionalMoves = p.build_possible_positional_moves(board)
	}
	return p.PossiblePositionalMoves
}

func (p Piece) build_possible_positional_moves(board *Board) (o []*models.Move) {
	var new_positions []int32
	for _, pos := range p.get_adjacent_positions() {
		if board.position_is_open(pos) {
			new_positions = append(new_positions, pos)
		}
	}
	return p.create_moves_from_new_positions(new_positions)
}

func (p Piece) create_moves_from_new_positions(new_positions []int32) (o []*models.Move) {
	for _, new_position := range new_positions {
		o = append(o, &models.Move{
			From: p.Position,
			To:   new_position,
		})
	}
	return
}

func (p Piece) get_adjacent_positions() (o []int32) {
	o = p.get_directional_adjacent_positions(true)
	if p.King {
		o = append(o, p.get_directional_adjacent_positions(false)...)
	}
	return
}

func (p Piece) get_directional_adjacent_positions(forward bool) (o []int32) {
	current_row := p.get_row()
	n := int32(-1)
	if Player(p.Player) == White {
		n = 1
	}
	f := int32(-1)
	if forward {
		f = 1
	}
	next_row := current_row + n*f
	if next_row > 0 && next_row < 8 {
		next_column_indexes := p.get_next_column_indexes(current_row, p.get_column())
		for _, column_index := range next_column_indexes {
			o = append(o, layout[next_row][column_index])
		}
	}
	return
}

func (p Piece) get_next_column_indexes(current_row, current_column int32) (o []int32) {
	column_indexes := []int32{current_column - 1, current_column}
	if current_row%2 == 0 {
		column_indexes[0] = current_column
		column_indexes[1] = current_column + 1
	}
	for _, column_index := range column_indexes {
		if column_index > 0 && column_index < Width {
			o = append(o, column_index)
		}
	}
	return
}

const (
	Width                 = 4
	Height                = 8
	RowsPerUserWithPieces = 3

	PositionCount      = Width * Height
	StartingPieceCount = Width * RowsPerUserWithPieces
)

type Layout [8][4]int32

var layout = InitialLayout()

func InitialLayout() Layout {
	return [8][4]int32{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
		{9, 10, 11, 12},
		{13, 14, 15, 16},
		{17, 18, 19, 20},
		{21, 22, 23, 24},
		{25, 26, 27, 28},
		{29, 30, 31, 32},
	}
}

func NewBoard() Board {
	b := Board{}
	b.set_starting_pieces()
	return b
}

func (b *Board) set_starting_pieces() {
	isWhite := func(po int32) bool {
		return po > 0 && po < StartingPieceCount+1
	}
	isBlack := func(po int32) bool {
		return po >= PositionCount-StartingPieceCount && po < PositionCount+1+1
	}
	var pieces []*models.Piece
	var id int32
	for _, row := range layout {
		for _, position := range row {
			id++
			var player Player
			if isWhite(position) {
				player = White
			} else if isBlack(position) {
				player = Black
			}
			pieces = append(pieces, &models.Piece{
				Id:       id,
				Player:   bool(player),
				Position: position,
			})
		}
	}
	b.Pieces = pieces
	b.resetPieces()
}
