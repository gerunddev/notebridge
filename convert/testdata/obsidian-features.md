---
id: abc12345-6789-def0-1234-56789abcdef0
title: Obsidian Features Test
aliases:
  - Obsidian Test
tags:
  - obsidian
  - features
---

# Obsidian-Specific Features

This file tests Obsidian-specific markdown features.

## Callouts

Obsidian supports 12 callout types (plus aliases):

> [!note]
> Basic note callout for general information.

> [!abstract]
> Abstract or summary callout.

> [!info]
> Additional informational callout.

> [!todo]
> Task or to-do callout.

> [!tip] Pro Tip
> Tip callout with custom title.

> [!success]
> Success or completion callout.

> [!question]
> Question or FAQ callout.

> [!warning]
> Warning callout for potential issues.

> [!failure]
> Failure or error state callout.

> [!danger]
> Danger or critical warning callout.

> [!bug]
> Bug report or issue callout.

> [!example]
> Example or code sample callout.

Aliases work too (summary, tldr, hint, important, check, done, help, faq, caution, attention, fail, missing, error).

## Embeds

Here's an embedded note:

![[Related Note]]

And an embedded image:

![[diagram.png]]

You can also embed specific headings:

![[Related Note#Introduction]]

## Heading Links

Link to a specific heading in another file:

[[Related Note#Tasks and Projects]]

Link to a heading in this file:

[[#Callouts]]

## Block References

This is a paragraph with a block ID. ^my-block-id

You can reference it like this: [[#^my-block-id]]

Or from another file: [[Obsidian Features Test#^my-block-id]]

## Inline Tags

This note discusses #productivity and #note-taking best practices.

You can also use nested tags like #obsidian/features and #obsidian/plugins.

## Wikilinks with Aliases

Link with display text: [[Related Note|Check out this note]]

Simple link: [[Related Note]]

## Conclusion

These features are commonly used in Obsidian vaults and need proper handling during conversion.
