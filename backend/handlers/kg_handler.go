package handlers

import (
	"net/http"
	"strconv"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"

	"github.com/gin-gonic/gin"
)

type KGNode struct {
	ID         int64       `json:"id"`
	EntityType string      `json:"entity_type"`
	EntityID   int64       `json:"entity_id"`
	Label      string      `json:"label"`
	Props      interface{} `json:"props"`
}
type KGEdge struct {
	ID      int64       `json:"id"`
	Src     int64       `json:"src"`
	Dst     int64       `json:"dst"`
	RelType string      `json:"rel_type"`
	Props   interface{} `json:"props"`
	Weight  float64     `json:"weight"`
}
type KGSubgraph struct {
	Center KGNode   `json:"center"`
	Nodes  []KGNode `json:"nodes"`
	Edges  []KGEdge `json:"edges"`
}

func GetKGNeighborhood(c *gin.Context) {
	etype := c.DefaultQuery("type", "asset")
	eidStr := c.Query("id")
	depthStr := c.DefaultQuery("depth", "1")
	if eidStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing id"})
		return
	}
	eid, _ := strconv.ParseInt(eidStr, 10, 64)
	depth, _ := strconv.Atoi(depthStr)
	if depth < 1 || depth > 3 {
		depth = 1
	}

	// center node
	var center KGNode
	err := database.Pool.QueryRow(c.Request.Context(),
		`SELECT id, entity_type, entity_id, label, props FROM kg_nodes WHERE entity_type=$1 AND entity_id=$2`, etype, eid).
		Scan(&center.ID, &center.EntityType, &center.EntityID, &center.Label, &center.Props)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "center node not found"})
		return
	}

	// 1-hop neighbors
	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT e.id, e.src_node_id, e.dst_node_id, e.rel_type, e.props, e.weight,
		       n2.id, n2.entity_type, n2.entity_id, n2.label, n2.props
		  FROM kg_edges e
		  JOIN kg_nodes n1 ON n1.id=e.src_node_id
		  JOIN kg_nodes n2 ON n2.id=e.dst_node_id
		 WHERE n1.id=$1
		UNION
		SELECT e.id, e.src_node_id, e.dst_node_id, e.rel_type, e.props, e.weight,
		       n1.id, n1.entity_type, n1.entity_id, n1.label, n1.props
		  FROM kg_edges e
		  JOIN kg_nodes n1 ON n1.id=e.src_node_id
		  JOIN kg_nodes n2 ON n2.id=e.dst_node_id
		 WHERE n2.id=$1
	`, center.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	nodesMap := map[int64]KGNode{center.ID: center}
	var edges []KGEdge
	for rows.Next() {
		var eid int64
		var src, dst int64
		var rel string
		var eprops interface{}
		var w float64
		var nid int64
		var nt string
		var ne int64
		var nl string
		var nprops interface{}
		if err := rows.Scan(&eid, &src, &dst, &rel, &eprops, &w, &nid, &nt, &ne, &nl, &nprops); err != nil {
			continue
		}
		nodesMap[nid] = KGNode{ID: nid, EntityType: nt, EntityID: ne, Label: nl, Props: nprops}
		edges = append(edges, KGEdge{ID: eid, Src: src, Dst: dst, RelType: rel, Props: eprops, Weight: w})
	}

	// (optional) expand to depth=2 dengan join berulang jika diperlukan

	// flatten
	var nodes []KGNode
	for _, n := range nodesMap {
		nodes = append(nodes, n)
	}

	c.JSON(http.StatusOK, KGSubgraph{
		Center: center,
		Nodes:  nodes,
		Edges:  edges,
	})
}
