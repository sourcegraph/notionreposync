# Plan

1. Get the path to the root to sync in Notion
2. Map the structure 
    - tree -> NotionPageID(s)
3. Walk the tree, starting at the root
    - Does this page exists already? 
        - Yes: synchronize it 
        - No: Create then synchronize it


## Usage

1. Create a page inside the database of syncronized doc repos.
2. Fill the `Repository` page property.
3. Click the share button and add the "GitHubDocSync" integration.

## Implementation notes

Once we're past the pages creation step on Notion, we're basically turning Markdown AST into Notion blocks. It's rather simple conceptually, but we have to 
keep in mind the differences. Whereas with HTML, everything is basically a tag, with Notion it's not that simple: 

- We deal in _Blocks_ and lists of _RichText_. 
    - A paragraph is a _Block_ composed of a list of _RichText_ elements.
    - `foo _bar_ baz` is just one block, with three _RichText_ elements.
- Notion doesn't have the concept of a "List _Block_", instead we just have "ListItem" _Blocks_, but the markdown parser we use does. 
    - So when we walk the AST, we cannot rely on the list node to know if we're about to create a list or exit from one, we instead rely on knowing if the list item is the first or last one.
