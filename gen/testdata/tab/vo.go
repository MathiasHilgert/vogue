package tab

// Title is the human-readable name of a tab, as the waiter typed it.
//vogue:string Title required trim lower min=1 max=120 nodigits

//vogue:int Covers min=1 max=200

// TabStatus is the lifecycle state of a tab.
//vogue:enum TabStatus open,in_progress,closed

// TabID identifies a tab across services.
//vogue:id TabID uuid7

//vogue:id InvoiceNumber int64
