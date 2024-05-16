1. Write an RFC describing the problem, data that will be added, and how Sourcegraph will use the data to make decisions. sourcegraph/bizops must be a required reviewer. Please include the following information RFC:
    - What are the exact data fields you're requesting to add?
    - What are the exact questions you're trying to answer with this new
    data? Why can't we use existing data to answer them?
    - How does the JSON payload look once those fields are added?
    
    The RFC should also include answers to these questions (if applicable):

    - Why was this particular metric/data chosen? What business problem does  collecting this address?
    - What specific product or engineering decisions will be made by having  this data?
    - Will this data be needed from every single installation, or only from a  select few?
    - Will it be needed forever, or only for a short time? If only for a  short time, what is the criteria and estimated timeline for removing the  data point(s)?
    - Have you considered alternatives? E.g., collecting this data from Sourcegraph.com, or adding a report for admins that we can request from some number of friendly customers?    
