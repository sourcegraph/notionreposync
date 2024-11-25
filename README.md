# NotionRepoSync

:warning: Work in progress, use at your own risk.

## Overview

_NotionRepoSync_ provides a CLI tool that can import in Notion markdown files from a given repository into Notion pages, while preserving the links between the imported documents. So if `a.md` links to `b.md`, if you browse the imported `a.md` in Notion, when you click on the link, you'll get sent to the imported `b.md` Notion page.

## What problem is this solving?

[Notion](https://notion.so) is a great tool for a company knowledge database, but comes with a few tradeoffs making it not the best solution for technnical/reference documentation. In particular, it's very common to keep the reference documentation of a given project versioned next to the code, as it enables to ship documentation changes along code changes and to review the whole thing in a single PR.

| Why would we want to bring reference documentation in Notion then?

Because it allows users to search through the internal compnany knowledge database and the reference documentation at the same time. It provides users a single entry point to crawl company knowledge and lift from users the requirement to know where something is documented in the first place.

Notion supports importing markdown, and there are tools to batch import markdown files, but they come with a big caveat: they do not handle internal links in between the imported pages.

_NotionRepoSync_ aims to address that particular problem and provides a CLI you can drop in your CI to automatically synchronize content in Notion. Becaues it keeps an index of the existing pages, across updates the page ids are stable, so no links are broken and folks can safely bookmark pages.

## Implementation

### Overview

Without going in details the general flow is the following:

1. CLI is called with a Notion page id that will hold the imported pages in a Notion database.
1. Walk the provided root folder containing markdown files.
1. Update the pages database in Notion, creating missing ones if needed. Original path is stored in row properties.
1. For each markdown file, we convert Markdown AST into Notion block and feed it to the API.

:warning: At this stage, we're only _appending_ to pages.

### What's done

- Creating or synchronzing the pages database.
- Parsing most of the common markdown blocks.
- Rendering and resolving links to corresponding Notion pages.
- Links to anything that isn't a markdown file is instead resolving to a code view.
- Usable as a library, for when you simply want to maintain specific content instead of integrating it with your CI.

### TODO

- Handling anchors
    - This requires to do post-processing, because anchors are unpredictable in Notion, as they point to block IDs, which cannot be known before their creation.
- Locking pages automatically
    - We don't want users to think they can edit the content in place, as it'll be overwritten on the next sync.
- Updating blocks instead of appending
    - Naive implementation should be to wipe all the existing blocks, and start over.
    - Better implementation would be to check if the next block matches what we expect, and just skip it if that's the case. Otherwise delete.
    - Even better might be to try to avoid as much possible to delete any block.
- If the document starts with a single `h1` and there are not `h1` in the content, then we can safely assume that this is the page title.
    - We can remove that `h1`, use it a title and turn all `h2` (and below) into `h1` (or `hN-1`) to compensate the fact that Notion only has three levels as opposed to markdown who has 6.

## Usage

`notionreposync --page-id <page id> --api-key <notion api key> <path to docs>`
* `page-id`: this is the uuid of the page, which you can get from the url ex. in the url `https://www.notion.so/sourcegraph/Docs-72707db6b0a74de2a13a6ec51b0e7f2e?pvs=4` the uuid is `72707db6b0a74de2a13a6ec51b0e7f2e?`
* `api-key`: Found at `Notion Markdown Importer` in 1pass
* `path to docs`: The path to the docs you want to sync. Generally if you are going to sync `sg/sg` you probably want the path `../sourcegraph/docs/dev`

### Notion Page requirements
1. Create a page inside the database of syncronized doc repos.
2. The page should have the following properties:
  1. 'Repository': text field
  2. 'LastSyncAt': date field
  3. 'LastSyncRev': text field
3. Click the share button and add the "GitHubDocSync" integration.

## Implementation notes

Once we're past the pages creation step on Notion, we're basically turning Markdown AST into Notion blocks. It's rather simple conceptually, but we have to
keep in mind the differences. Whereas with HTML, everything is basically a tag, with Notion it's not that simple:

- We deal in _Blocks_ and lists of _RichText_.
    - A paragraph is a _Block_ composed of a list of _RichText_ elements.
    - `foo _bar_ baz` is just one block, with three _RichText_ elements.
- Notion doesn't have the concept of a "List _Block_", instead we just have "ListItem" _Blocks_, but the markdown parser we use does.
    - So when we walk the AST, we cannot rely on the list node to know if we're about to create a list or exit from one, we instead rely on knowing if the list item is the first or last one.
