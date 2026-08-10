(() => {
  if (customElements.get("mate-d2")) return;

  class MateD2 extends HTMLElement {
    constructor() {
      super();
      this.attachShadow({mode:"open"});
      this._graph = {nodes:[],edges:[],source:""};
      this._selected = "";
      this._renderToken = 0;
      this._view = null;
      this._d2 = null;
    }
    connectedCallback() {
      this.shadowRoot.innerHTML = `<style>:host{display:block;min-height:38rem;border:1px solid #d8e0e9;border-radius:14px;overflow:hidden;background:#fbfcfe;box-shadow:0 10px 30px rgba(15,23,42,.06);font-family:Inter,ui-sans-serif,system-ui}.bar{display:flex;align-items:center;justify-content:space-between;padding:.55rem .7rem;border-bottom:1px solid #e3e8ef;background:rgba(255,255,255,.9);color:#64748b;font-size:.75rem}.actions{display:flex;gap:.4rem}button{appearance:none;border:1px solid #d8e0e9;border-radius:7px;background:#fff;color:#334155;padding:.38rem .65rem;font:600 .72rem inherit;cursor:pointer}.viewport{height:38rem;overflow:hidden;touch-action:none;cursor:grab}.viewport.dragging{cursor:grabbing}.loading{display:grid;place-items:center;height:38rem;color:#64748b}.viewport svg{display:block;width:100%;height:100%;max-width:none}.node-target{cursor:pointer}.node-target.selected .shape>*{stroke:#b4532a!important;stroke-width:4!important}</style><div class="bar"><span>Rendered with D2</span><div class="actions"><button type="button" data-fit>Fit</button></div></div><div class="viewport"><div class="loading">Preparing D2 diagram…</div></div>`;
      this.shadowRoot.querySelector("[data-fit]").onclick = () => this._fit();
      this._render();
    }
    disconnectedCallback() {
      this._renderToken++;
      this._drag = null;
    }
    set graph(value) {
      this._graph = value || {nodes:[],edges:[],source:""};
      this._render();
    }
    get graph() { return this._graph; }
    async _renderer() {
      if (this._d2) return this._d2;
      const {D2} = await import("/arch/static/d2-0.1.33.js");
      this._d2 = new D2();
      return this._d2;
    }
    async _render() {
      const viewport = this.shadowRoot?.querySelector(".viewport");
      if (!viewport || !this._graph.source) return;
      const token = ++this._renderToken;
      viewport.innerHTML = '<div class="loading">Rendering D2 diagram…</div>';
      try {
        const d2 = await this._renderer();
        const compiled = await d2.compile(this._graph.source,{layout:"elk"});
        const output = await d2.render(compiled.diagram,{...compiled.renderOptions,noXMLTag:true,pad:36,scale:1});
        if (token !== this._renderToken || !this.isConnected) return;
        const template = document.createElement("template");
        template.innerHTML = output.trim();
        template.content.querySelectorAll("script,foreignObject").forEach(element => element.remove());
        const svg = template.content.querySelector("svg");
        if (!svg) throw new Error("D2 did not return an SVG");
        viewport.replaceChildren(svg);
        this._svg = svg;
        this._bind();
        this._fit();
      } catch (error) {
        if (token !== this._renderToken) return;
        viewport.innerHTML = '<div class="loading">The D2 diagram could not be rendered.</div>';
        console.error("mate-d2 render failed",error);
      }
    }
    _bind() {
      const svg = this._svg;
      const viewport = this.shadowRoot.querySelector(".viewport");
      for (const node of this._graph.interactive === false ? [] : (this._graph.nodes || [])) {
        const className = btoa(`&#34;${node.id}&#34;`);
        const target = svg.querySelector(`.${CSS.escape(className)}`);
        if (!target) continue;
        target.classList.add("node-target");
        target.setAttribute("role","button");
        target.setAttribute("tabindex","0");
        if (node.id === this._selected) target.classList.add("selected");
        const select = event => {
          event.preventDefault();
          event.stopPropagation();
          this._selected = node.id;
          svg.querySelectorAll(".node-target.selected").forEach(item => item.classList.remove("selected"));
          target.classList.add("selected");
          this.dispatchEvent(new CustomEvent("node-select",{detail:{id:node.id},bubbles:true}));
        };
        target.addEventListener("click",select);
        target.addEventListener("keydown",event => {
          if (event.key === "Enter" || event.key === " ") select(event);
        });
      }
      if (this._graph.interactive !== false) {
        const edgeTargets = [...svg.querySelectorAll('[class*="edge"]')];
        edgeTargets.forEach((target,index) => {
          const edge = this._graph.edges?.[index];
          if (!edge) return;
          target.classList.add("edge-target");
          target.setAttribute("role","button");
          target.addEventListener("click", event => {
            event.preventDefault();
            event.stopPropagation();
            this.dispatchEvent(new CustomEvent("edge-select", {detail:{edge},bubbles:true}));
          });
        });
      }
      svg.addEventListener("wheel",event => {
        event.preventDefault();
        const factor = event.deltaY > 0 ? 1.12 : .88;
        const point = this._point(event.clientX,event.clientY);
        const nextW = this._view.w*factor;
        const nextH = this._view.h*factor;
        this._view.x = point.x-(point.x-this._view.x)*factor;
        this._view.y = point.y-(point.y-this._view.y)*factor;
        this._view.w = nextW;
        this._view.h = nextH;
        this._applyView();
      },{passive:false});
      svg.addEventListener("pointerdown",event => {
        if (event.target.closest(".node-target")) return;
        svg.setPointerCapture(event.pointerId);
        viewport.classList.add("dragging");
        this._drag = {x:event.clientX,y:event.clientY,vx:this._view.x,vy:this._view.y};
      });
      svg.addEventListener("pointermove",event => {
        if (!this._drag) return;
        this._view.x = this._drag.vx-(event.clientX-this._drag.x)*this._view.w/svg.clientWidth;
        this._view.y = this._drag.vy-(event.clientY-this._drag.y)*this._view.h/svg.clientHeight;
        this._applyView();
      });
      const end = () => { this._drag=null; viewport.classList.remove("dragging"); };
      svg.addEventListener("pointerup",end);
      svg.addEventListener("pointercancel",end);
    }
    _point(clientX,clientY) {
      const rect = this._svg.getBoundingClientRect();
      return {
        x:this._view.x+(clientX-rect.left)/rect.width*this._view.w,
        y:this._view.y+(clientY-rect.top)/rect.height*this._view.h
      };
    }
    _fit() {
      if (!this._svg) return;
      const parts = (this._svg.getAttribute("viewBox") || `0 0 ${this._svg.getAttribute("width")} ${this._svg.getAttribute("height")}`).split(/\s+/).map(Number);
      this._naturalView = {x:parts[0]||0,y:parts[1]||0,w:parts[2]||1000,h:parts[3]||600};
      this._view = {...this._naturalView};
      this._applyView();
    }
    _applyView() {
      if (this._svg && this._view) this._svg.setAttribute("viewBox",`${this._view.x} ${this._view.y} ${this._view.w} ${this._view.h}`);
    }
  }
  customElements.define("mate-d2",MateD2);
})();
