# Quakefile Neovim Plugin

Syntax highlighting and filetype support for Quakefile in Neovim.

## Features

- Syntax highlighting for:
  - Tasks, namespaces, and file namespaces
  - Dependencies and arguments
  - Comments and documentation
  - Variables and expressions (`{{...}}`)
  - Command substitution (backticks)
  - Special command prefixes (`@` for silent, `-` for continue on error)
  - Strings (single, double, and multi-line)
  
- Auto-indent support for nested blocks
- Comment formatting support
- Code folding for tasks and namespaces

## Installation

### Using a plugin manager

#### lazy.nvim

```lua
{
  "quake",
  dir = "/path/to/quake/nvim",
  ft = "quakefile",
}
```

#### packer.nvim

```lua
use {
  '/path/to/quake/nvim',
  ft = 'quakefile'
}
```

### Manual installation

Copy the plugin files to your Neovim configuration directory:

```bash
cp -r nvim/* ~/.config/nvim/
```

Or create symlinks:

```bash
ln -s /path/to/quake/nvim/ftdetect ~/.config/nvim/ftdetect/quakefile.vim
ln -s /path/to/quake/nvim/syntax ~/.config/nvim/syntax/quakefile.vim
ln -s /path/to/quake/nvim/indent ~/.config/nvim/indent/quakefile.vim
ln -s /path/to/quake/nvim/ftplugin ~/.config/nvim/ftplugin/quakefile.vim
```

## File Detection

The plugin automatically detects the following files as Quakefile:
- `Quakefile` (exact name)
- `*.quake` files
- `*_Quakefile` files (e.g., `api_Quakefile`, `frontend_Quakefile`)

## Usage

Once installed, the plugin will automatically activate when you open a Quakefile.

### Folding

Code folding is enabled by default. Use these commands:
- `za` - Toggle fold
- `zo` - Open fold
- `zc` - Close fold
- `zR` - Open all folds
- `zM` - Close all folds

### Comments

Use `#` for comments. The plugin supports comment formatting with `gq` and comment toggling if you have a comment plugin installed.