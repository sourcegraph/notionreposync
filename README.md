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
