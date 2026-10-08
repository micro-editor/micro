Copy and paste are essential features in micro but can be
confusing to get right especially when running micro over SSH
because there are multiple methods. This help document will explain
the various methods for copying and pasting, how they work,
and the best methods for doing so over SSH.

# OSC 52 (terminal clipboard)

If possible, setting the `clipboard` option to `terminal` will give
best results because it will work over SSH and locally. However, there
is limited support among terminal emulators for the terminal clipboard
(which uses the OSC 52 protocol to communicate clipboard contents).
Here is a list of terminal emulators and their status:

* `Kitty`: supported, but only writing is enabled by default. To enable
   reading, add `read-primary` and `read-clipboard` to the
   `clipboard_control` option.

* `iTerm2`: supported — fully since 3.5.0, only copying (writing to
   clipboard) in earlier versions.. Must be enabled in
   `Preferences->General-> Selection->Applications in terminal may access clipboard`.
   Additionally, consider setting `Allow sending of clipboard content` to `Always`.

* `st`: supported.

* `rxvt-unicode`: not natively supported, but there is a Perl extension
   [here](https://anti.teamidiot.de/static/nei/*/Code/urxvt/).

* `xterm`: supported, but disabled by default. It can be enabled by putting
   the following in `.Xresources` or `.Xdefaults`:
   `XTerm*disallowedWindowOps: 20,21,SetXprop`.

* `gnome-terminal`: does not support OSC 52.

* `alacritty`: supported. Since 0.13.0, reading has been disabled by default.
   To reenable it, set the `terminal.osc52` option to `CopyPaste`.

* `foot`: supported.

* `wezterm`: only copying (writing to clipboard) is supported.


**Summary:** If you want copy and paste to work over SSH, then you
should set `clipboard` to `terminal`, and make sure your terminal
supports OSC 52.

# Pasting

## Recommendations (TL;DR)

The recommended method of pasting is the following:

* If you are not working over SSH, use the micro keybinding (`Ctrl-v`
  by default) to perform pastes. If on Linux, install `xclip` or
  `xsel` beforehand.

* If you are working over SSH, use the terminal keybinding
  (`Ctrl-Shift-v` or `Command-v`) to perform pastes. If your terminal
  does not support bracketed paste, when performing a paste first
  enable the `paste` option, and when finished disable the option.

## Micro paste events

Micro is an application that runs within the terminal. This means
that the terminal sends micro events, such as key events, mouse
events, resize events, and paste events. Micro's default keybinding
for paste is `Ctrl-v`. This means that when micro receives the key
event saying `Ctrl-v` has been pressed from the terminal, it will
attempt to access the system clipboard and effect a paste. The
system clipboard will be accessed through `pbpaste` on MacOS
(installed by default), `xclip` or `xsel` on Linux (these
applications must be installed by the user) or a system call on
Windows.

## Terminal paste events

For certain keypresses, the terminal will not send an event to
micro and will instead do something itself. In this document,
such keypresses will be called "terminal keybindings." Often
there will be a terminal keybinding for pasting and copying. On
MacOS these are Command-v and Command-c and on Linux `Ctrl-Shift-v`
and `Ctrl-Shift-c`. When the terminal keybinding for paste is
executed, your terminal will access the system clipboard, and send
micro either a paste event or a list of key events (one key for each
character in the paste), depending on whether or not your terminal
supports sending paste events (called bracketed paste).

If your terminal supports bracketed paste, then it will send a paste
event and everything will work well. However, if your terminal
sends a list of key events, this can cause issues because micro
will think you manually entered each character and may add closing
brackets or automatic indentation, which will mess up the pasted
text. To avoid this, you can temporarily enable the `paste` option
while you perform the paste. When paste option is on, micro will
aggregate lists of multiple key events into larger paste events.
It is a good idea to disable the `paste` option during normal use
as occasionally if you are typing quickly, the terminal will send
the key events as lists of characters that were in fact manually
entered.

## Pasting over SSH

When working over SSH, micro is running on the remote machine and
your terminal is running on your local machine. Therefore if you
would like to paste, using `Ctrl-v` (micro's keybinding) will not
work because when micro attempts to access the system clipboard,
it will access the remote machine's clipboard rather than the local
machine's clipboard. On the other hand, the terminal keybinding
for paste will access your local clipboard and send the text over
the network as a paste event, which is what you want.

# Copying

# Recommendations (TL;DR)

The recommended method of copying is the following:

* If you are not working over SSH, use the micro keybinding (`Ctrl-c` by
  default) to perform copies. If on Linux, install `xclip` or `xsel`
  beforehand.

* If you are working over SSH, use the terminal keybinding
  (`Ctrl-Shift-c` or `Command-c`) to perform copies. You must first disable
  the `mouse` option to perform a terminal selection, and you may wish
  to disable line numbers and diff indicators (`ruler` and `diffgutter`
  options) and close other splits. This method will only be able to copy
  characters that are displayed on the screen (you will not be able to
  copy more than one page's worth of characters).

Copying follows a similar discussion to the one above about pasting.
The primary difference is before performing a copy, the application
doing the copy must be told what text needs to be copied.

Micro has a keybinding (`Ctrl-c`) for copying and will access the system
clipboard to perform the copy. The text that micro will copy into is
the text that is currently selected in micro (usually such text is
displayed with a white background). When the `mouse` option is enabled,
the mouse can be used to select text, as well as other keybindings,
such as ShiftLeft, etc...

The terminal also has a keybinding (`Ctrl-Shift-c` or `Command-c`) to perform
a copy, and the text that it copies is the text selected by the terminal's
selection (*not* micro's selection). To select text with the terminal
selection, micro's mouse support must first be disabled by turning the
`mouse` option off. The terminal, unlike micro, has no sense of different
buffers/splits and what the different characters being displayed are. This
means that for copying multiple lines using the terminal selection, you
should first disable line numbers and diff indicators (turn off the `ruler`
and `diffgutter` options), otherwise they might be part of your selection
and copied.
