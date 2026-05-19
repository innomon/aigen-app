# Account Context

An `Account` represents a node in the chart of accounts. Accounts can be of different types, such as Asset, Liability, Equity, Income, or Expense.
It can also be a "Group" account (which cannot have transactions directly posted to it) or a "Ledger" account (which can have transactions).

When creating an account, ensure it is linked to the correct Parent Account to maintain the tree structure.