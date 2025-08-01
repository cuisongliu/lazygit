package controllers

import (
	"strings"

	"github.com/jesseduffield/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

type PatchExplorerControllerFactory struct {
	c *ControllerCommon
}

func NewPatchExplorerControllerFactory(c *ControllerCommon) *PatchExplorerControllerFactory {
	return &PatchExplorerControllerFactory{
		c: c,
	}
}

func (self *PatchExplorerControllerFactory) Create(context types.IPatchExplorerContext) *PatchExplorerController {
	return &PatchExplorerController{
		baseController: baseController{},
		c:              self.c,
		context:        context,
	}
}

type PatchExplorerController struct {
	baseController
	c *ControllerCommon

	context types.IPatchExplorerContext
}

func (self *PatchExplorerController) Context() types.Context {
	return self.context
}

func (self *PatchExplorerController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{
		{
			Tag:     "navigation",
			Key:     opts.GetKey(opts.Config.Universal.PrevItemAlt),
			Handler: self.withRenderAndFocus(self.HandlePrevLine),
		},
		{
			Tag:     "navigation",
			Key:     opts.GetKey(opts.Config.Universal.PrevItem),
			Handler: self.withRenderAndFocus(self.HandlePrevLine),
		},
		{
			Tag:     "navigation",
			Key:     opts.GetKey(opts.Config.Universal.NextItemAlt),
			Handler: self.withRenderAndFocus(self.HandleNextLine),
		},
		{
			Tag:     "navigation",
			Key:     opts.GetKey(opts.Config.Universal.NextItem),
			Handler: self.withRenderAndFocus(self.HandleNextLine),
		},
		{
			Tag:         "navigation",
			Key:         opts.GetKey(opts.Config.Universal.RangeSelectUp),
			Handler:     self.withRenderAndFocus(self.HandlePrevLineRange),
			Description: self.c.Tr.RangeSelectUp,
		},
		{
			Tag:         "navigation",
			Key:         opts.GetKey(opts.Config.Universal.RangeSelectDown),
			Handler:     self.withRenderAndFocus(self.HandleNextLineRange),
			Description: self.c.Tr.RangeSelectDown,
		},
		{
			Key:         opts.GetKey(opts.Config.Universal.PrevBlock),
			Handler:     self.withRenderAndFocus(self.HandlePrevHunk),
			Description: self.c.Tr.PrevHunk,
		},
		{
			Key:     opts.GetKey(opts.Config.Universal.PrevBlockAlt),
			Handler: self.withRenderAndFocus(self.HandlePrevHunk),
		},
		{
			Key:         opts.GetKey(opts.Config.Universal.NextBlock),
			Handler:     self.withRenderAndFocus(self.HandleNextHunk),
			Description: self.c.Tr.NextHunk,
		},
		{
			Key:     opts.GetKey(opts.Config.Universal.NextBlockAlt),
			Handler: self.withRenderAndFocus(self.HandleNextHunk),
		},
		{
			Key:         opts.GetKey(opts.Config.Universal.ToggleRangeSelect),
			Handler:     self.withRenderAndFocus(self.HandleToggleSelectRange),
			Description: self.c.Tr.ToggleRangeSelect,
		},
		{
			Key:         opts.GetKey(opts.Config.Main.ToggleSelectHunk),
			Handler:     self.withRenderAndFocus(self.HandleToggleSelectHunk),
			Description: self.c.Tr.ToggleSelectHunk,
			DescriptionFunc: func() string {
				if state := self.context.GetState(); state != nil && state.SelectingHunk() {
					return self.c.Tr.SelectLineByLine
				}
				return self.c.Tr.SelectHunk
			},
			Tooltip:         self.c.Tr.ToggleSelectHunkTooltip,
			DisplayOnScreen: true,
		},
		{
			Tag:         "navigation",
			Key:         opts.GetKey(opts.Config.Universal.PrevPage),
			Handler:     self.withRenderAndFocus(self.HandlePrevPage),
			Description: self.c.Tr.PrevPage,
		},
		{
			Tag:         "navigation",
			Key:         opts.GetKey(opts.Config.Universal.NextPage),
			Handler:     self.withRenderAndFocus(self.HandleNextPage),
			Description: self.c.Tr.NextPage,
		},
		{
			Tag:         "navigation",
			Key:         opts.GetKey(opts.Config.Universal.GotoTop),
			Handler:     self.withRenderAndFocus(self.HandleGotoTop),
			Description: self.c.Tr.GotoTop,
		},
		{
			Tag:         "navigation",
			Key:         opts.GetKey(opts.Config.Universal.GotoBottom),
			Description: self.c.Tr.GotoBottom,
			Handler:     self.withRenderAndFocus(self.HandleGotoBottom),
		},
		{
			Tag:     "navigation",
			Key:     opts.GetKey(opts.Config.Universal.GotoTopAlt),
			Handler: self.withRenderAndFocus(self.HandleGotoTop),
		},
		{
			Tag:     "navigation",
			Key:     opts.GetKey(opts.Config.Universal.GotoBottomAlt),
			Handler: self.withRenderAndFocus(self.HandleGotoBottom),
		},
		{
			Tag:     "navigation",
			Key:     opts.GetKey(opts.Config.Universal.ScrollLeft),
			Handler: self.withRenderAndFocus(self.HandleScrollLeft),
		},
		{
			Tag:     "navigation",
			Key:     opts.GetKey(opts.Config.Universal.ScrollRight),
			Handler: self.withRenderAndFocus(self.HandleScrollRight),
		},
		{
			Key:         opts.GetKey(opts.Config.Universal.CopyToClipboard),
			Handler:     self.withLock(self.CopySelectedToClipboard),
			Description: self.c.Tr.CopySelectedTextToClipboard,
		},
	}
}

func (self *PatchExplorerController) GetMouseKeybindings(opts types.KeybindingsOpts) []*gocui.ViewMouseBinding {
	return []*gocui.ViewMouseBinding{
		{
			ViewName: self.context.GetViewName(),
			Key:      gocui.MouseLeft,
			Handler: func(opts gocui.ViewMouseBindingOpts) error {
				if self.isFocused() {
					return self.withRenderAndFocus(self.HandleMouseDown)()
				}

				self.c.Context().Push(self.context, types.OnFocusOpts{
					ClickedWindowName:  self.context.GetWindowName(),
					ClickedViewLineIdx: opts.Y,
				})

				return nil
			},
		},
		{
			ViewName: self.context.GetViewName(),
			Key:      gocui.MouseLeft,
			Modifier: gocui.ModMotion,
			Handler: func(gocui.ViewMouseBindingOpts) error {
				return self.withRenderAndFocus(self.HandleMouseDrag)()
			},
		},
	}
}

func (self *PatchExplorerController) HandlePrevLine() error {
	before := self.context.GetState().GetSelectedViewLineIdx()
	self.context.GetState().CycleSelection(false)
	after := self.context.GetState().GetSelectedViewLineIdx()

	if self.context.GetState().SelectingLine() {
		checkScrollUp(self.context.GetViewTrait(), self.c.UserConfig(), before, after)
	}

	return nil
}

func (self *PatchExplorerController) HandleNextLine() error {
	before := self.context.GetState().GetSelectedViewLineIdx()
	self.context.GetState().CycleSelection(true)
	after := self.context.GetState().GetSelectedViewLineIdx()

	if self.context.GetState().SelectingLine() {
		checkScrollDown(self.context.GetViewTrait(), self.c.UserConfig(), before, after)
	}

	return nil
}

func (self *PatchExplorerController) HandlePrevLineRange() error {
	s := self.context.GetState()

	s.CycleRange(false)

	return nil
}

func (self *PatchExplorerController) HandleNextLineRange() error {
	s := self.context.GetState()

	s.CycleRange(true)

	return nil
}

func (self *PatchExplorerController) HandlePrevHunk() error {
	self.context.GetState().SelectPreviousHunk()

	return nil
}

func (self *PatchExplorerController) HandleNextHunk() error {
	self.context.GetState().SelectNextHunk()

	return nil
}

func (self *PatchExplorerController) HandleToggleSelectRange() error {
	self.context.GetState().ToggleStickySelectRange()

	return nil
}

func (self *PatchExplorerController) HandleToggleSelectHunk() error {
	self.context.GetState().ToggleSelectHunk()

	return nil
}

func (self *PatchExplorerController) HandleScrollLeft() error {
	self.context.GetViewTrait().ScrollLeft()

	return nil
}

func (self *PatchExplorerController) HandleScrollRight() error {
	self.context.GetViewTrait().ScrollRight()

	return nil
}

func (self *PatchExplorerController) HandlePrevPage() error {
	self.context.GetState().AdjustSelectedLineIdx(-self.context.GetViewTrait().PageDelta())

	return nil
}

func (self *PatchExplorerController) HandleNextPage() error {
	self.context.GetState().AdjustSelectedLineIdx(self.context.GetViewTrait().PageDelta())

	return nil
}

func (self *PatchExplorerController) HandleGotoTop() error {
	self.context.GetState().SelectTop()

	return nil
}

func (self *PatchExplorerController) HandleGotoBottom() error {
	self.context.GetState().SelectBottom()

	return nil
}

func (self *PatchExplorerController) HandleMouseDown() error {
	self.context.GetState().SelectNewLineForRange(self.context.GetViewTrait().SelectedLineIdx())

	return nil
}

func (self *PatchExplorerController) HandleMouseDrag() error {
	self.context.GetState().DragSelectLine(self.context.GetViewTrait().SelectedLineIdx())

	return nil
}

func (self *PatchExplorerController) CopySelectedToClipboard() error {
	selected := self.context.GetState().PlainRenderSelected()

	self.c.LogAction(self.c.Tr.Actions.CopySelectedTextToClipboard)
	if err := self.c.OS().CopyToClipboard(dropDiffPrefix(selected)); err != nil {
		return err
	}

	return nil
}

// Removes '+' or '-' from the beginning of each line in the diff string, except
// when both '+' and '-' lines are present, or diff header lines, in which case
// the diff is returned unchanged. This is useful for copying parts of diffs to
// the clipboard in order to paste them into code.
func dropDiffPrefix(diff string) string {
	lines := strings.Split(strings.TrimRight(diff, "\n"), "\n")

	const (
		PLUS int = iota
		MINUS
		CONTEXT
		OTHER
	)

	linesByType := lo.GroupBy(lines, func(line string) int {
		switch {
		case strings.HasPrefix(line, "+"):
			return PLUS
		case strings.HasPrefix(line, "-"):
			return MINUS
		case strings.HasPrefix(line, " "):
			return CONTEXT
		}
		return OTHER
	})

	hasLinesOfType := func(lineType int) bool { return len(linesByType[lineType]) > 0 }

	keepPrefix := hasLinesOfType(OTHER) || (hasLinesOfType(PLUS) && hasLinesOfType(MINUS))
	if keepPrefix {
		return diff
	}

	return strings.Join(lo.Map(lines, func(line string, _ int) string { return line[1:] + "\n" }), "")
}

func (self *PatchExplorerController) isFocused() bool {
	return self.c.Context().Current().GetKey() == self.context.GetKey()
}

func (self *PatchExplorerController) withRenderAndFocus(f func() error) func() error {
	return self.withLock(func() error {
		if err := f(); err != nil {
			return err
		}

		self.context.RenderAndFocus()
		return nil
	})
}

func (self *PatchExplorerController) withLock(f func() error) func() error {
	return func() error {
		self.context.GetMutex().Lock()
		defer self.context.GetMutex().Unlock()

		if self.context.GetState() == nil {
			return nil
		}

		return f()
	}
}
