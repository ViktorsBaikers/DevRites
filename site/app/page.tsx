import Nav from "@/components/Nav";
import Hero from "@/components/Hero";
import Trust from "@/components/Trust";
import Contrast from "@/components/Contrast";
import Bento from "@/components/Bento";
import Pipeline from "@/components/Pipeline";
import Anywhere from "@/components/Anywhere";
import Faq from "@/components/Faq";
import Install from "@/components/Install";
import Footer from "@/components/Footer";

export default function Home() {
  return (
    <>
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-50 focus:rounded-lg focus:bg-surface focus:px-4 focus:py-2 focus:text-ink"
      >
        Skip to content
      </a>
      <Nav />
      <main id="main" className="w-full max-w-full overflow-x-clip">
        <Hero />
        <Trust />
        <Pipeline />
        <Contrast />
        <Bento />
        <Anywhere />
        <Faq />
        <Install />
      </main>
      <Footer />
    </>
  );
}
