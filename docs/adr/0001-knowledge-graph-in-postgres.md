# Store the knowledge graph in PostgreSQL, not a graph database

A Course's knowledge graph is small (hundreds of Concepts at most), and the queries we need — all Prerequisites of a Concept, the weakest Prerequisite on the path to it — are answerable with recursive SQL or by loading one Course's graph into memory in the Go service. Adding Neo4j would mean a second datastore to keep consistent with learner and question data in PostgreSQL, for no capability we need. Revisit only if graphs grow by orders of magnitude or we need graph-algorithm workloads across Courses.
