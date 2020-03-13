import { Injectable, EventEmitter } from '@angular/core';
import * as d3 from 'd3';
import { ForceLink } from 'd3-force';

@Injectable()
export class D3ViewService{
  forceSimulation: any;
  extendNode: EventEmitter<number> = new EventEmitter<number>();

  constructor() {}

  updatePath(nodes: any, links: any) {
    let ee = this.extendNode;

    let svg = d3.select("svg").select("g");

    let div = d3.select("body").append("div")
      .attr("class", "detail-tooltip")
      .style("opacity", 0);

    svg.selectAll('*').remove();

    let width: number = 750;
    let height: number = 600;

    // https://github.com/d3/d3-scale-chromatic
    let color: Array<string> = ['#9bd0c6','#ffffb6','#bbb8d8','#eb8773','#88aed0','#f3b768','#bbdc71','#f4cee4','#d8d8d8','#b182ba','#d1e9c5','#fdee78','#a3adbd'];

    let simulation = d3.forceSimulation()
      .force("link", d3.forceLink().id(function(d) { return d['id']; }))
      .force("charge", d3.forceManyBody().strength(-5000).distanceMin(10).distanceMax(1000))
      .force("center", d3.forceCenter(width / 2, height / 2))
      .force("y", d3.forceY(0.001))
      .force("x", d3.forceX(0.001));

    let defs = svg.append('svg:defs');
    defs.append('svg:marker')
      .attr('id', 'end-arrow')
      .attr('viewBox', '0 0 20 20')
      .attr('refX', "23")
      .attr('refY', "3")
      .attr('markerWidth', "10")
      .attr('markerHeight', "10")
      .attr('markerUnits', "strokeWidth")
      .attr('orient', 'auto')
      .append('svg:path')
      .attr('d', 'M0,0 L0,6 L9,3 z')
      .attr('fill', '#7a8084');

    let link = svg.append("g")
      .attr("class", "links")
      .selectAll("line")
      .data(links)
      .enter().append("line")
      //.attr("stroke", "#999")
      .attr('stroke', function (d) {
        return color[d['group']];
      })
      .attr('marker-end', 'url(#end-arrow)')
      .attr("stroke-opacity", "0.6")
      .attr("stroke-width", "3")
      .attr("stroke-dasharray", function (d) { return d['group'] == '0' ? '0' : '3'});

    link.exit().remove();

    let node = svg.selectAll(".node")
      .data(nodes)
      .enter().append("g")
      .attr("class", "node")
      .call(d3.drag()
        .on("start", function (d) {
          if (!d3.event.active) simulation.alphaTarget(0.3).restart();
          d['fx'] = d['x'];
          d['fy'] = d['y'];
        })
        .on("drag", function (d) {
          d['fx'] = d3.event.x;
          d['fy'] = d3.event.y;
        }))
      .on("mouseover", function(d) {

        if(d['cpu'] != '') {
          div.transition()
            .duration(200)
            .style("opacity", 0.9);
          div.html('<strong>' + d['cpu'] + '</br>' + d['memory'] + '</br>' + d['disk'] + '</strong>')
            .style("left", (d3.event.pageX + 10) + "px")
            .style("top", (d3.event.pageY - 28) + "px");
        }
      })
      .on("mouseout", function(d) {
        div.transition()
          .duration(10)
          .style("opacity", 0);
      })

      .on("click", function(d) {
        // simulation.stop();
        if (d['type'] == 'API' && d['active'] != 'extended') {
          d['active'] = 'extended';
          // ee.emit(d['id']);
        }
      });

    // .on("end", function (d) {
    //   if (!d3.event.active) simulation.alphaTarget(0);
    //   d['fx'] = null;
    //   d['fy'] = null;
    // }));

    node.append('circle')
      .attr('r', 25)
      .attr('fill', function (d) {
        return color[d['group']];
      })
      .style('stroke', '#6980a3')
      .style('stroke-width', 2)
      .transition()
      .duration(1000)
      .on("start", function repeat(d) {
        if (String(d['active']).toLowerCase() == 'stopped') {
          d3.active(this)
            .attr("r", 25)
            .transition()
            .attr("r", 30)
            .attr('fill', 'red')
            .transition()
            .on("start", repeat);
        }
        if (d['type'] == 'API' && d['active'] != 'extended') {
          d3.active(this)
            .style('cursor', 'hand');
        }
      });



    node.append("text")
      .attr("dx", function (d) {
        return d['type'] == 'App' || d['type'] == 'API'  ? "-12" : "-21";
      })
      .attr("dy", 3)
      .style('fill', 'white')
      .style('display', 'inline')
      .style("font-size", "13px")
      .text(function (d) {
        return d['type'];
      });

    node.append("text")
      .attr("dx", 30)
      .attr("dy", 11)
      .style('fill', 'black')
      .style('display', 'inline')
      .style("font-size", "18px")
      .style('text-shadow', '0 3px 0 #fff, 3px 0 0 #fff, 0 -3px 0 #fff, -3px 0 0 #fff')
      .text(function (d) {
        return d['name'];
      });

    /*
    let text_node = node.append("text")
      .attr("dx", 30)
      .attr("dy", 27)
      .style('fill', 'gray')
      .style("font-size", "13px");

    text_node
      .append("tspan")
      .attr("x", 0)
      .attr("y", 2)
      .text(function(d){
        return d['cpu'];
      });
    text_node
      .append("tspan")
      .attr("x", 30)
      .attr("y", 43)
      .text(function(d){
        return d['memory'];
      });
    text_node
      .append("tspan")
      .attr("x", 30)
      .attr("y", 59)
      .text(function(d){
        return d['disk'];
      });
    */

    node.exit().remove();

    simulation
      .nodes(nodes)
      .on("tick", function()  {
        link
          .attr("x1", function(d) { return d['source']['x']; })
          .attr("y1", function(d) { return d['source']['y']; })
          .attr("x2", function(d) { return d['target']['x']; })
          .attr("y2", function(d) { return d['target']['y']; });
        node.attr("transform", function (d) {
          return "translate(" + d['x'] + "," + d['y'] + ")";
        });
      });

    simulation.force<ForceLink<any, any>>("link").links(links);

    simulation.restart();

    this.forceSimulation = simulation;
  }

  stopSvg() {
    this.forceSimulation.stop();
  }

}
